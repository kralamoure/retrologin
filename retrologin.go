// Package retrologin implements an unofficial login server for Dofus Retro.
package retrologin

import (
	"errors"
	"net"
	"time"

	"github.com/happybydefault/logging"
	"github.com/kralamoure/dofus/dofussvc"
	"github.com/kralamoure/retro/retrosvc"
)

type Config struct {
	Addr        string
	ConnTimeout time.Duration
	TicketDur   time.Duration
	Dofus       *dofussvc.Service
	Retro       *retrosvc.Service
	Logger      logging.Logger
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
	if c.Retro == nil {
		return nil, errors.New("nil retro service")
	}
	if c.Logger == nil {
		c.Logger = logging.Noop{}
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
		retro:              c.Retro,
		sessions:           make(map[*session]struct{}),
		sessionByAccountId: make(map[string]*session),
	}
	return s, nil
}
