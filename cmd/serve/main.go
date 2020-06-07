package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kralamoure/d1/repository/postgres"
	"github.com/kralamoure/d1/service/login"
	"github.com/kralamoure/d1proto/msgsvr"
	"go.uber.org/zap"

	"github.com/kralamoure/d1login"
)

var logger *zap.SugaredLogger

type config struct {
	host               string
	port               string
	postgresConnString string
	sharedKey          string
	dev                bool
}

func main() {
	os.Exit(run())
}

func run() (exitCode int) {
	cfg := config{}

	flag.StringVar(&cfg.host, "host", "0.0.0.0", "host of the listener")
	flag.StringVar(&cfg.port, "port", "5555", "port of the listener")
	flag.StringVar(&cfg.postgresConnString, "db", "postgresql://user:password@host/database",
		"postgres connection string, either in URL or DSN format")
	flag.StringVar(&cfg.sharedKey, "key", "", "32 characters long shared key")
	flag.BoolVar(&cfg.dev, "dev", false, "sets logging to development mode")

	flag.Parse()

	var tmpLogger *zap.Logger
	if cfg.dev {
		tmp, err := zap.NewDevelopment()
		if err != nil {
			log.Println(err)
			return 1
		}
		tmpLogger = tmp
	} else {
		tmp, err := zap.NewProduction()
		if err != nil {
			log.Println(err)
			return 1
		}
		tmpLogger = tmp
	}
	logger = tmpLogger.Sugar()
	defer logger.Sync()

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		logger.Debugf("received signal: %s", sig)
		signal.Stop(sigs)
		cancelCtx()
	}()

	logger.Info("connecting to postgres")
	db, err := postgres.NewDB(context.Background(), cfg.postgresConnString)
	if err != nil {
		logger.Error(err)
		return 1
	}
	defer db.Close()

	svcCfg := d1login.Config{
		Login:     login.NewService(db),
		Logger:    logger,
		SharedKey: []byte(cfg.sharedKey),
	}

	s := d1login.NewServer(svcCfg)

	laddr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(cfg.host, cfg.port))
	if err != nil {
		logger.Errorf("could not resolve TCP address: %s", err)
		return 1
	}
	var ln *net.TCPListener
	if tmp, err := net.ListenTCP("tcp", laddr); err != nil {
		logger.Errorf("error while listening for connections: %s", err)
		return 1
	} else {
		ln = tmp
	}
	defer ln.Close()
	logger.Infof("listening for connections on %s", ln.Addr())

	conns := make(chan *net.TCPConn, 0)
	go func() {
		for {
			conn, err := ln.AcceptTCP()
			if err != nil {
				if ctx.Err() == nil {
					logger.Errorf("error while accepting connections: %s", err)
					cancelCtx()
				}
				break
			}
			conns <- conn
		}
	}()

	wg := sync.WaitGroup{}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			var err error
			select {
			case <-ticker.C:
				wg.Add(1)
				oldHostsData := s.HostsData()

				err = s.UpdateHostsData()
				if err != nil {
					err = fmt.Errorf("could not update hosts data: %w", err)
					break
				}

				hostsData := s.HostsData()

				if hostsData == oldHostsData {
					break
				}

				hosts := &msgsvr.AccountHosts{}
				hosts.Deserialize(s.HostsData())

				for sess := range s.Sessions() {
					if sess.Status() != d1login.SessionStatusIdle {
						continue
					}
					s.SendPacketMsg(sess.Conn, hosts)
				}
			case <-ctx.Done():
				return
			}

			if err != nil {
				logger.Error(err)
				cancelCtx()
				wg.Done()
				return
			}
			wg.Done()
		}
	}()

LOOP:
	for {
		select {
		case <-ctx.Done():
			logger.Debug(ctx.Err())
			ln.Close()
			break LOOP
		case conn := <-conns:
			wg.Add(1)
			go func() {
				err := handleSession(s, ctx, conn)
				if err != nil {
					if !(errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) || d1login.IsClosedConnError(err)) {
						logger.Debugw(fmt.Sprintf("error while handling session: %s", err.Error()),
							"address", conn.RemoteAddr(),
						)
					}
				}
				wg.Done()
			}()
		}
	}

	wg.Wait()

	return 0
}
