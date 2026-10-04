// Command homebase runs the Homebase server.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"homebase/internal/api"
	"homebase/internal/config"
	"homebase/internal/webui"
)

const usage = `Usage: homebase <command>

Commands:
  serve    run the server
`

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(os.Args[1:], os.Getenv, os.Stderr, log); err != nil {
		log.Error("exit", "err", err)
		os.Exit(1)
	}
}

func run(args []string, getenv func(string) string, stderr io.Writer, log *slog.Logger) error {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return errors.New("no command given")
	}
	switch args[0] {
	case "serve":
		cfg, err := config.Load(getenv)
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return serve(ctx, cfg, log)
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(stderr, usage)
		return nil
	default:
		_, _ = fmt.Fprint(stderr, usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func serve(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return fmt.Errorf("create data folder: %w", err)
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.New(log, webui.Handler()),
		ReadHeaderTimeout: 10 * time.Second,
		// Generous enough for a 25 MB upload on a slow phone connection.
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  2 * time.Minute,
		ErrorLog:     slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}

	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr, "data", cfg.DataDir)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
