package retrologin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/kralamoure/retroproto/msgsvr"
	"go.uber.org/zap"
)

type Server struct {
	logger *zap.SugaredLogger

	mu      sync.Mutex
	clients map[*client]struct{}
}

func NewServer(logger *zap.SugaredLogger) *Server {
	return &Server{
		clients: make(map[*client]struct{}),
		logger:  logger,
	}
}

func (s *Server) Serve(ctx context.Context, l *net.TCPListener) error {
	var wg sync.WaitGroup
	defer wg.Wait()

	defer func() {
		if err := l.Close(); err != nil {
			s.logger.Errorf("could not close listener: %s", err)
		} else {
			s.logger.Infof("listener closed: %s", l.Addr())
		}
	}()

	errCh := make(chan error, 1)
	conns := make(chan *net.TCPConn)
	go func() {
		for {
			conn, err := l.AcceptTCP()
			if err != nil {
				errCh <- fmt.Errorf("could not accept connection: %w", err)
			}
			conns <- conn
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case conn := <-conns:
			c := s.newClient(conn)

			wg.Add(1)
			go func() {
				defer wg.Done()
				s.handleClient(ctx, c)
			}()
		}
	}
}

func (s *Server) handleClient(ctx context.Context, c *client) {
	s.trackClient(c, true)
	defer s.trackClient(c, false)

	s.logger.Infof("new client: %s", c)
	defer s.logger.Infof("client disconnected: %s", c)

	r := bufio.NewReader(c.conn)

	errCh := make(chan error, 1)
	packets := make(chan string)
	go func() {
		for {
			packet, err := r.ReadString('\x00')
			if err != nil {
				errCh <- fmt.Errorf("error when reading: %w", err)
				return
			}
			packet = strings.TrimSuffix(packet[:len(packet)-1], "\n")

			select {
			case packets <- packet:
			case <-ctx.Done():
			}
		}
	}()

	c.sendMsg(msgsvr.AksHelloConnect{Salt: "tospfquxqwdbqibcckgxxoitwsunpzgp"})

	logger := s.logger.With("client", c)
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errCh:
			if !errors.Is(err, io.EOF) {
				logger.Info(err)
			}
			return
		case packet := <-packets:
			err := c.handlePacket(packet)
			if err != nil {
				logger.Infof("could not handle packet: %s", err)
			}
		}
	}
}

func (s *Server) trackClient(c *client, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if add {
		s.clients[c] = struct{}{}
	} else {
		delete(s.clients, c)
	}
}

func (s *Server) newClient(conn *net.TCPConn) *client {
	return &client{
		logger: s.logger.With("client", conn.RemoteAddr().String()),
		server: s,
		conn:   conn,
	}
}
