package main

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync"

	"github.com/kralamoure/d1proto/msgsvr"

	"github.com/kralamoure/d1login"
)

func handleSession(s *d1login.Server, ctx context.Context, conn *net.TCPConn) error {
	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

	sess, err := d1login.NewSession(conn)
	if err != nil {
		return err
	}

	s.AddSession(sess)
	defer func() {
		s.DeleteSession(sess)
	}()

	errChan := make(chan error, 1)

	wg := &sync.WaitGroup{}

	r := bufio.NewReader(sess.Conn)
	go func() {
		for {
			data, err := r.ReadString('\x00')
			if err != nil {
				errChan <- err
				return
			}
			data = strings.Trim(data, " \t\r\n\x00")
			if data == "" {
				continue
			}

			wg.Add(1)
			err = handlePacketData(s, sess, data)
			wg.Done()
			if err != nil {
				errChan <- err
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		s.HandlePacketsQueue(ctx, sess.Conn, sess.PktCh)
		wg.Done()
	}()

	s.SendPacketMsg(sess.Conn, &msgsvr.AksHelloConnect{Salt: sess.Salt})

	var e error
	select {
	case <-ctx.Done():
		e = ctx.Err()
	case err := <-errChan:
		e = err
		cancelCtx()
	}

	wg.Wait()

	return e
}
