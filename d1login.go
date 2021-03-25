package d1login

import (
	"errors"
	"net"
	"time"

	"github.com/happybydefault/logger"
	"github.com/kralamoure/d1/d1svc"
	"github.com/kralamoure/dofus/dofussvc"
)

type Config struct {
	Addr        string
	ConnTimeout time.Duration
	TicketDur   time.Duration
	Dofus       *dofussvc.Service
	D1          *d1svc.Service
	Logger      logger.Logger
}

func NewServer(c Config) (*Server, error) {
	if c.ConnTimeout <= 0 {
		c.ConnTimeout = 30 * time.Minute
	}
	if c.TicketDur <= 0 {
		c.TicketDur = 20 * time.Second
	}
	if c.Dofus == nil {
		return nil, errors.New("nil dofus service")
	}
	if c.D1 == nil {
		return nil, errors.New("nil d1 service")
	}
	if c.Logger == nil {
		c.Logger = logger.Noop{}
	}
	addr, err := net.ResolveTCPAddr("tcp4", c.Addr)
	if err != nil {
		return nil, err
	}
	s := &Server{
		logger:             c.Logger,
		addr:               addr,
		connTimeout:        c.ConnTimeout,
		ticketDur:          c.TicketDur,
		dofus:              c.Dofus,
		d1:                 c.D1,
		sessions:           make(map[*session]struct{}),
		sessionByAccountId: make(map[string]*session),
	}
	return s, nil
}
