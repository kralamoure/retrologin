package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/trace"
	"sync"
	"syscall"

	"github.com/kralamoure/d1postgres"
	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/kralamoure/d1login"
)

const version = "v0.0.0"

var (
	printVersion bool
	debug        bool
	addr         string
	pgConnString string
)

var logger *zap.Logger

func main() {
	os.Exit(run())
}

func run() int {
	err := loadVars()
	if err != nil {
		if errors.Is(err, pflag.ErrHelp) {
			return 0
		}
		log.Println(err)
		return 2
	}

	if printVersion {
		fmt.Println(version)
		return 0
	}

	if debug {
		traceFile, err := os.Create("trace.out")
		if err != nil {
			log.Println(err)
			return 1
		}
		defer traceFile.Close()
		err = trace.Start(traceFile)
		if err != nil {
			log.Println(err)
			return 1
		}
		defer trace.Stop()

		logger, err = zap.NewDevelopment()
		if err != nil {
			log.Println(err)
			return 1
		}
	} else {
		logger, err = zap.NewProduction()
		if err != nil {
			log.Println(err)
			return 1
		}
	}
	defer logger.Sync()

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	errCh := make(chan error)

	repo, err := d1postgres.NewDB(ctx, pgConnString)
	if err != nil {
		log.Println(err)
		return 1
	}

	svr, err := d1login.NewServer(d1login.ServerConfig{
		Addr:   addr,
		Repo:   repo,
		Logger: logger.Named("server"),
	})
	if err != nil {
		logger.Error("could not make login server", zap.Error(err))
		return 1
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := svr.ListenAndServe(ctx)
		if err != nil {
			select {
			case errCh <- fmt.Errorf("error while listening and serving: %w", err):
			case <-ctx.Done():
			}
		}
	}()

	select {
	case sig := <-sigCh:
		logger.Info("received signal",
			zap.String("signal", sig.String()),
		)
	case err := <-errCh:
		logger.Error(err.Error())
		return 1
	case <-ctx.Done():
	}
	return 0
}

func loadVars() error {
	flags := pflag.NewFlagSet("d1login", pflag.ContinueOnError)
	flags.BoolVarP(&printVersion, "version", "v", false, "Print version")
	flags.BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	flags.StringVarP(&addr, "address", "a", "0.0.0.0:5555", "server listener's address")
	flags.StringVarP(&pgConnString, "postgres", "p", "postgresql://user:password@host/database", "PostgreSQL connection string")
	flags.SortFlags = false
	return flags.Parse(os.Args)
}
