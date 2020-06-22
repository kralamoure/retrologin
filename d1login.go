package d1login

import (
	"errors"
	"net"

	"github.com/kralamoure/d1/service/login"
	"github.com/kralamoure/d1proto/msgsvr"
	"go.uber.org/zap"
)

type Config struct {
	Addr    string
	Service *login.Service
	Logger  *zap.Logger
}

func NewServer(c Config) (*Server, error) {
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
		logger: c.Logger,
		addr:   addr,
		svc:    c.Service,
	}
	s.hosts.Store(msgsvr.AccountHosts{})
	return s, nil
}
