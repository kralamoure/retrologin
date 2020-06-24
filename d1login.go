package d1login

import (
	"errors"
	"net"
	"time"

	"github.com/kralamoure/d1/service/login"
	"go.uber.org/zap"
)

const Version = "v0.5.0"

type Config struct {
	Addr      string
	TicketDur time.Duration
	Service   *login.Service
	Logger    *zap.Logger
}

func NewServer(c Config) (*Server, error) {
	if c.TicketDur <= 0 {
		c.TicketDur = 20 * time.Second
	}
	if c.Service == nil {
		return nil, errors.New("nil service")
	}
	if c.Logger == nil {
		c.Logger = zap.NewNop()
	}
	addr, err := net.ResolveTCPAddr("tcp4", c.Addr)
	if err != nil {
		return nil, err
	}
	s := &Server{
		logger:    c.Logger,
		addr:      addr,
		ticketDur: c.TicketDur,
		svc:       c.Service,
	}
	return s, nil
}
