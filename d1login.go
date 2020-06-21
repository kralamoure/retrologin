package d1login

import (
	"errors"
	"net"

	"github.com/kralamoure/d1"
	"go.uber.org/zap"
)

type Config struct {
	Addr   string
	Repo   d1.Repo
	Logger *zap.Logger
}

func NewServer(c Config) (*Server, error) {
	if c.Repo == nil {
		return nil, errors.New("nil repository")
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
		repo:   c.Repo,
	}
	return s, nil
}
