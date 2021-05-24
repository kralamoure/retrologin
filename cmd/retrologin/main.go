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

	"github.com/happybydefault/logging"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/kralamoure/dofus/dofussvc"
	"github.com/kralamoure/dofuspg"
	"github.com/kralamoure/retro/retrosvc"
	"github.com/kralamoure/retropg"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"

	"github.com/kralamoure/retrologin"
)

const (
	programName        = "retrologin"
	programDescription = "retrologin is an unofficial login server for Dofus Retro."
	programMoreInfo    = "https://github.com/kralamoure/retrologin"
)

var (
	printHelp    bool
	debug        bool
	serverAddr   string
	connTimeout  time.Duration
	pgConnString string
)

var (
	flagSet *pflag.FlagSet
	logger  *zap.SugaredLogger
)

func main() {
	l := log.New(os.Stderr, "", 0)

	initFlagSet()
	err := flagSet.Parse(os.Args)
	if err != nil {
		l.Println(err)
		os.Exit(2)
	}

	if printHelp {
		fmt.Println(help(flagSet.FlagUsages()))
		return
	}

	if debug {
		tmp, err := zap.NewDevelopment()
		if err != nil {
			l.Println(err)
			os.Exit(1)
		}
		logger = tmp.Sugar()
	} else {
		tmp, err := zap.NewProduction()
		if err != nil {
			l.Println(err)
			os.Exit(1)
		}
		logger = tmp.Sugar()
	}

	err = run()
	if err != nil {
		logger.Error(err)
		os.Exit(1)
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

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := pgxpool.ParseConfig(pgConnString)
	if err != nil {
		return err
	}
	pool, err := pgxpool.ConnectConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	dofusDb, err := dofuspg.NewDb(pool)
	if err != nil {
		return err
	}

	retroDb, err := retropg.NewDb(pool)
	if err != nil {
		return err
	}

	dofusSvc, err := dofussvc.NewService(dofusDb)
	if err != nil {
		return err
	}

	retroSvc, err := retrosvc.NewService(retrosvc.Config{
		Storer: retroDb,
	})
	if err != nil {
		return err
	}

	svr, err := retrologin.NewServer(retrologin.Config{
		Addr:        serverAddr,
		ConnTimeout: connTimeout,
		Dofus:       dofusSvc,
		Retro:       retroSvc,
		Logger:      logging.Named("server", logger),
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	errCh := make(chan error)

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
	flagSet = pflag.NewFlagSet("retrologin", pflag.ContinueOnError)
	flagSet.BoolVarP(&printHelp, "help", "h", false, "Print usage information")
	flagSet.BoolVarP(&debug, "debug", "d", false, "Enable debug mode")
	flagSet.StringVarP(&serverAddr, "address", "a", "0.0.0.0:5555", "Server listener address")
	flagSet.StringVarP(&pgConnString, "postgres", "p", "postgresql://user:password@host/database", "PostgreSQL connection string")
	flagSet.DurationVarP(&connTimeout, "timeout", "t", 30*time.Minute, "Connection timeout")
	flagSet.SortFlags = false
}
