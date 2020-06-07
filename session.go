package d1login

import (
	"net"
	"sync"
	"time"

	"github.com/kralamoure/d1proto/msgcli"
)

const (
	SessionStatusExpectingVersion SessionStatus = iota
	SessionStatusExpectingCredential
	SessionStatusExpectingFirstQueuePosition
	SessionStatusIdle
)

type SessionStatus int

type Session struct {
	Conn       *net.TCPConn
	Salt       string
	PktCh      chan QueuePacket
	Version    msgcli.AccountVersion
	Credential msgcli.AccountCredential
	AccountId  int
	LastAccess time.Time
	LastIP     string

	mu     sync.Mutex
	status SessionStatus
}

func NewSession(conn *net.TCPConn) (*Session, error) {
	salt, err := RandomSalt(32)
	if err != nil {
		return nil, err
	}

	return &Session{
		Conn:  conn,
		PktCh: make(chan QueuePacket),
		Salt:  salt,
	}, nil
}

func (s *Session) Status() SessionStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.status
}

func (s *Session) SetStatus(status SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status = status
}
