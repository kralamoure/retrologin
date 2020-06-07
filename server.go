package d1login

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/kralamoure/d1/service/login"
	"github.com/kralamoure/d1proto/msgsvr"
	"github.com/kralamoure/d1proto/typ"
	"github.com/o1egl/paseto"
	"go.uber.org/zap"
)

type Server struct {
	Logger    *zap.SugaredLogger
	Login     login.Service
	sharedKey []byte

	mu        sync.Mutex
	hostsData string
	sessions  map[*Session]struct{}
}

func NewServer(cfg Config) *Server {
	svr := &Server{
		Logger:    cfg.Logger,
		Login:     cfg.Login,
		sharedKey: cfg.SharedKey,

		sessions: make(map[*Session]struct{}),
	}
	return svr
}

func (s *Server) HostsData() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.hostsData
}

func (s *Server) SetHostsData(hosts string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.hostsData = hosts
}

func (s *Server) UpdateHostsData() error {
	gameservers, err := s.Login.GameServers()
	if err != nil {
		return err
	}

	var sli []typ.AccountHostsHost
	for _, gameserver := range gameservers {
		host := typ.AccountHostsHost{
			Id:         gameserver.Id,
			State:      int(gameserver.State),
			Completion: int(gameserver.Completion),
			CanLog:     true,
		}
		sli = append(sli, host)
	}
	sort.Slice(sli, func(i, j int) bool { return sli[i].Id < sli[j].Id })

	msg := msgsvr.AccountHosts{Value: sli}
	hosts, err := msg.Serialized()
	if err != nil {
		return err
	}

	s.SetHostsData(hosts)

	return nil
}

func (s *Server) DeleteSessionByAccountId(accountId int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for sess := range s.sessions {
		if sess.AccountId == accountId {
			s.deleteSessionLocked(sess)
			break
		}
	}
}

func (s *Server) DeleteSession(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.Conn.Close()
	_, ok := s.sessions[sess]
	if !ok {
		return
	}
	delete(s.sessions, sess)
	s.Logger.Debugw("deleted session",
		"address", sess.Conn.RemoteAddr(),
	)
}

func (s *Server) deleteSessionLocked(sess *Session) {
	sess.Conn.Close()
	_, ok := s.sessions[sess]
	if !ok {
		return
	}
	delete(s.sessions, sess)
	s.Logger.Debugw("deleted session",
		"address", sess.Conn.RemoteAddr(),
	)
}

func (s *Server) AddSession(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sess] = struct{}{}
	s.Logger.Debugw("added session",
		"address", sess.Conn.RemoteAddr(),
	)
}

func (s *Server) Sessions() map[*Session]struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.sessions
}

func (s *Server) TokenData(accountId, gameserverId int, lastAccess time.Time, lastIP string) (string, error) {
	now := time.Now()
	token := paseto.JSONToken{
		Subject:    fmt.Sprint(accountId),
		Expiration: now.Add(5 * time.Second),
	}
	token.Set("serverId", fmt.Sprintf("%d", gameserverId))
	token.Set("lastAccess", fmt.Sprintf("%d", lastAccess.Unix()))
	token.Set("lastIP", lastIP)

	data, err := paseto.NewV2().Encrypt(s.sharedKey, token, nil)
	if err != nil {
		return "", err
	}

	return data, nil
}
