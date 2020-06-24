package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/trace"
	"sync"
	"syscall"
	"time"

	"github.com/kralamoure/d1/service/login"
	"github.com/kralamoure/d1postgres"
	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/kralamoure/d1login"
)

const (
	programName        = "d1login"
	programDescription = "d1login is a login server for Dofus 1."
)

var (
	printHelp    bool
	printVersion bool
	debug        bool
	addr         string
	connTimeout  time.Duration
	pgConnString string
)

var (
	flagSet *pflag.FlagSet
	logger  *zap.Logger
)

func main() {
	initFlagSet()
	err := flagSet.Parse(os.Args)
	if err != nil {
		log.Println(err)
		os.Exit(2)
	}

	if printHelp {
		fmt.Println(help(flagSet.FlagUsages()))
		return
	}
	if printVersion {
		fmt.Println(d1login.Version)
		return
	}

	if debug {
		logger, err = zap.NewDevelopment()
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		logger, err = zap.NewProduction()
		if err != nil {
			log.Fatalln(err)
		}
	}

	err = run()
	if err != nil {
		logger.Fatal(err.Error())
	}
}

func run() error {
	defer logger.Sync()

	if debug {
		traceFile, err := os.Create("trace.out")
		if err != nil {
			return err
		}
		defer traceFile.Close()
		err = trace.Start(traceFile)
		if err != nil {
			return err
		}
		defer trace.Stop()
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	errCh := make(chan error)

	logger.Info("connecting to db")
	db, err := d1postgres.NewDB(ctx, pgConnString)
	if err != nil {
		return err
	}
	defer db.Close()

	svc, err := login.NewService(login.Config{
		Repo:   db,
		Logger: logger.Named("service"),
	})
	if err != nil {
		return err
	}

	svr, err := d1login.NewServer(d1login.Config{
		Addr:        addr,
		ConnTimeout: connTimeout,
		Service:     svc,
		Logger:      logger.Named("server"),
	})
	if err != nil {
		return err
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
		return err
	case <-ctx.Done():
	}
	return nil
}

func help(flagUsages string) string {
	return fmt.Sprintf("Usage: %s [options]\n\n%s\n\nOptions:\n%s", programName, programDescription, flagUsages)
}

func initFlagSet() {
	flagSet = pflag.NewFlagSet("d1login", pflag.ContinueOnError)
	flagSet.BoolVarP(&printHelp, "help", "h", false, "Print usage information")
	flagSet.BoolVarP(&printVersion, "version", "v", false, "Print version")
	flagSet.BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	flagSet.StringVarP(&addr, "address", "a", "0.0.0.0:5555", "Server listener address")
	flagSet.StringVarP(&pgConnString, "postgres", "p", "postgresql://user:password@host/database", "PostgreSQL connection string")
	flagSet.DurationVarP(&connTimeout, "timeout", "t", 30*time.Minute, "Connection timeout")
	flagSet.SortFlags = false
}
