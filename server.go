package d1login

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/happybydefault/logger"
	"github.com/kralamoure/d1/d1svc"
	"github.com/kralamoure/d1proto/msgsvr"
	"github.com/kralamoure/d1proto/typ"
	"github.com/kralamoure/dofus/dofussvc"
	"go.uber.org/atomic"
)

type Server struct {
	logger      logger.Logger
	addr        *net.TCPAddr
	connTimeout time.Duration
	ticketDur   time.Duration
	dofus       *dofussvc.Service
	d1          *d1svc.Service

	mu                 sync.Mutex
	ln                 *net.TCPListener
	sessions           map[*session]struct{}
	sessionByAccountId map[string]*session

	hosts atomic.String
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	hosts, err := s.fetchHosts(ctx)
	if err != nil {
		return err
	}
	s.hosts.Store(hosts)

	ln, err := net.ListenTCP("tcp4", s.addr)
	if err != nil {
		return err
	}
	defer func() {
		ln.Close()
		s.logger.Infow("stopped listening",
			"address", ln.Addr().String(),
		)
	}()
	s.logger.Infow("listening",
		"address", ln.Addr().String(),
	)
	s.ln = ln

	errCh := make(chan error)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.watchTickets(ctx, 1*time.Second)
		if err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.watchHosts(ctx, 1*time.Second)
		if err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := s.acceptLoop(ctx)
		if err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (s *Server) controlAccount(accountId string, sess *session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentSess, ok := s.sessionByAccountId[accountId]
	if ok {
		currentSess.conn.Close()
		return errors.New("already logged in")
	}

	s.sessionByAccountId[accountId] = sess

	return nil
}

func (s *Server) acceptLoop(ctx context.Context) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		conn, err := s.ln.AcceptTCP()
		if err != nil {
			return err
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := s.handleClientConn(ctx, conn)
			if err != nil {
				var isTimeout bool
				netErr, ok := err.(net.Error)
				if ok {
					isTimeout = netErr.Timeout()
				}
				if !(isTimeout ||
					errors.Is(err, io.EOF) ||
					errors.Is(err, context.Canceled) ||
					errors.Is(err, errInvalidRequest)) {
					s.logger.Errorw(fmt.Errorf("error while handling client connection: %w", err).Error(),
						"client_address", conn.RemoteAddr().String(),
					)
				}
			}
		}()
	}
}

func (s *Server) handleClientConn(ctx context.Context, conn *net.TCPConn) error {
	salt, err := randomSalt(32)
	if err != nil {
		return err
	}

	sess := &session{
		svr:  s,
		conn: conn,
		salt: salt,
	}

	s.trackSession(sess, true)
	defer s.trackSession(sess, false)

	var wg sync.WaitGroup
	defer wg.Wait()

	defer func() {
		conn.Close()
		s.logger.Infow("client disconnected",
			"client_address", conn.RemoteAddr().String(),
		)
	}()
	s.logger.Info("client connected",
		"client_address", conn.RemoteAddr().String(),
	)

	err = conn.SetKeepAlivePeriod(1 * time.Minute)
	if err != nil {
		return err
	}
	err = conn.SetKeepAlive(true)
	if err != nil {
		return err
	}
	err = conn.SetDeadline(time.Now().UTC().Add(s.connTimeout))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := sess.receivePackets(ctx)
		if err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
		}
	}()

	sess.sendMessage(msgsvr.AksHelloConnect{Salt: sess.salt})

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) watchHosts(ctx context.Context, d time.Duration) error {
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hosts, err := s.fetchHosts(ctx)
			if err != nil {
				return err
			}
			if hosts != s.hosts.Load() {
				s.hosts.Store(hosts)
				var m msgsvr.AccountHosts
				err := m.Deserialize(hosts)
				if err != nil {
					return err
				}
				s.sendUpdatedHosts(m)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Server) watchTickets(ctx context.Context, d time.Duration) error {
	ticker := time.NewTicker(d)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count, err := s.deleteOldTickets(ctx)
			if err != nil {
				return err
			}
			if count > 0 {
				s.logger.Debugw("deleted old tickets",
					"count", count,
				)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Server) sendUpdatedHosts(hosts msgsvr.AccountHosts) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for sess := range s.sessions {
		if sess.status.Load() != statusIdle {
			continue
		}
		sess.sendMessage(hosts)
	}
}

func (s *Server) fetchHosts(ctx context.Context) (string, error) {
	gameServers, err := s.d1.GameServers(ctx)
	if err != nil {
		return "", err
	}

	var sli []typ.AccountHostsHost
	for _, gameServer := range gameServers {
		host := typ.AccountHostsHost{
			Id:         gameServer.Id,
			State:      int(gameServer.State),
			Completion: int(gameServer.Completion),
			CanLog:     true,
		}
		sli = append(sli, host)
	}
	sort.Slice(sli, func(i, j int) bool { return sli[i].Id < sli[j].Id })

	m := msgsvr.AccountHosts{Value: sli}
	hosts, err := m.Serialized()
	if err != nil {
		return "", err
	}
	return hosts, nil
}

func (s *Server) deleteOldTickets(ctx context.Context) (count int, err error) {
	return s.d1.DeleteTickets(ctx, time.Now().UTC().Add(-s.ticketDur))
}

func (s *Server) trackSession(sess *session, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if add {
		s.sessions[sess] = struct{}{}
	} else {
		delete(s.sessionByAccountId, sess.accountId)
		delete(s.sessions, sess)
	}
}
