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
	"go.uber.org/zap/buffer"

	"github.com/kralamoure/d1login"
)

const (
	programName        = "d1login"
	programDescription = "d1login is a login server for Dofus 1."
	programMoreInfo    = "https://github.com/kralamoure/d1login"
)

var (
	printHelp    bool
	printVersion bool
	debug        bool
	serverAddr   string
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
		fmt.Fprintln(os.Stderr, err)
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

	l := log.New(os.Stderr, "", 0)
	if debug {
		logger, err = zap.NewDevelopment()
		if err != nil {
			l.Fatalln(err)
		}
	} else {
		logger, err = zap.NewProduction()
		if err != nil {
			l.Fatalln(err)
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
		Addr:        serverAddr,
		ConnTimeout: connTimeout,
		Service:     svc,
		Logger:      logger.Named("server"),
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	defer wg.Wait()

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

	var selErr error
	select {
	case sig := <-sigCh:
		signal.Stop(sigCh)
		logger.Info("received signal",
			zap.String("signal", sig.String()),
		)
	case err := <-errCh:
		selErr = err
	case <-ctx.Done():
	}
	cancel()
	return selErr
}

func help(flagUsages string) string {
	buf := &buffer.Buffer{}
	fmt.Fprintf(buf, "%s\n\n", programDescription)
	fmt.Fprintf(buf, "Find more information at: %s\n\n", programMoreInfo)
	fmt.Fprint(buf, "Options:\n")
	fmt.Fprintf(buf, "%s\n", flagUsages)
	fmt.Fprintf(buf, "Usage: %s [options]", programName)
	return buf.String()
}

func initFlagSet() {
	flagSet = pflag.NewFlagSet("d1login", pflag.ContinueOnError)
	flagSet.BoolVarP(&printHelp, "help", "h", false, "Print usage information")
	flagSet.BoolVarP(&printVersion, "version", "v", false, "Print version")
	flagSet.BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	flagSet.StringVarP(&serverAddr, "address", "a", "0.0.0.0:5555", "Server listener address")
	flagSet.StringVarP(&pgConnString, "postgres", "p", "postgresql://user:password@host/database", "PostgreSQL connection string")
	flagSet.DurationVarP(&connTimeout, "timeout", "t", 30*time.Minute, "Connection timeout")
	flagSet.SortFlags = false
}
