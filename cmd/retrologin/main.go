package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/kralamoure/retrologin"
)

func main() {
	var development bool
	var address string

	flag.BoolVar(&development, "d", false, "Enable development mode")
	flag.StringVar(&address, "a", ":5555", "TCP address to listen at")
	flag.Parse()

	var logger *zap.SugaredLogger
	if l, err := newLogger(development); err != nil {
		log.Printf("could not create logger: %s", err)
	} else {
		logger = l.Sugar()
	}
	defer logger.Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh,
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	go func() {
		sig := <-sigCh
		signal.Stop(sigCh)
		logger.Infof("signal received: %s", sig)
		cancel()
	}()

	if err := serve(ctx, logger, address); err != nil {
		logger.Error(err)
	}
}

func serve(ctx context.Context, logger *zap.SugaredLogger, address string) error {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return fmt.Errorf("could not parse address: %w", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return fmt.Errorf("could not listen: %w", err)
	}
	logger.Infof("listening at %s", listener.Addr())

	svr := retrologin.NewServer(logger)

	if err = svr.Serve(ctx, listener); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("error while serving: %w", err)
	}

	return nil
}

func newLogger(development bool) (*zap.Logger, error) {
	var config zap.Config
	if development {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("could not build logger: %w", err)
	}

	return logger, nil
}
