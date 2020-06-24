package d1login

import (
	"context"
	"errors"
	"io"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/kralamoure/d1/filter"
	"github.com/kralamoure/d1/service/login"
	"github.com/kralamoure/d1proto/msgsvr"
	"github.com/kralamoure/d1proto/typ"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type Server struct {
	logger    *zap.Logger
	addr      *net.TCPAddr
	ticketDur time.Duration
	svc       *login.Service

	mu       sync.Mutex
	ln       *net.TCPListener
	sessions map[*session]struct{}

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
		s.logger.Info("stopped listening",
			zap.String("address", ln.Addr().String()),
		)
	}()
	s.logger.Info("listening",
		zap.String("address", ln.Addr().String()),
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
			if err != nil && !(errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) || errors.Is(err, errEndOfService)) {
				s.logger.Debug("error while handling client connection",
					zap.Error(err),
					zap.String("client_address", conn.RemoteAddr().String()),
				)
			}
		}()
	}
}

func (s *Server) handleClientConn(ctx context.Context, conn *net.TCPConn) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	defer func() {
		conn.Close()
		s.logger.Debug("client disconnected",
			zap.String("client_address", conn.RemoteAddr().String()),
		)
	}()
	s.logger.Debug("client connected",
		zap.String("client_address", conn.RemoteAddr().String()),
	)

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

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error)

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := sess.receivePkts(ctx)
		if err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
		}
	}()

	sess.sendMsg(msgsvr.AksHelloConnect{Salt: sess.salt})

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
			err := s.deleteOldTickets(ctx)
			if err != nil {
				return err
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
		sess.sendMsg(hosts)
	}
}

func (s *Server) fetchHosts(ctx context.Context) (string, error) {
	gameServers, err := s.svc.GameServers(ctx)
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

func (s *Server) deleteOldTickets(ctx context.Context) error {
	return s.svc.DeleteTickets(ctx, filter.TicketCreatedLT(time.Now().Add(-s.ticketDur)))
}

func (s *Server) trackSession(sess *session, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if add {
		if s.sessions == nil {
			s.sessions = make(map[*session]struct{})
		}
		s.sessions[sess] = struct{}{}
	} else {
		delete(s.sessions, sess)
	}
}
