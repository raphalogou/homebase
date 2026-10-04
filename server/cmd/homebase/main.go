// Command homebase runs the Homebase server.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // settings.tz must load on hosts without a zoneinfo database

	"homebase/internal/api"
	"homebase/internal/auth"
	"homebase/internal/config"
	"homebase/internal/db"
	"homebase/internal/files"
	"homebase/internal/migrate"
	"homebase/internal/push"
	"homebase/internal/sched"
	"homebase/internal/store"
	"homebase/internal/syncer"
	"homebase/internal/webui"
	"homebase/migrations"
)

const usage = `Usage: homebase <command>

Commands:
  serve              run the server
  hash-passphrase    read a passphrase and print HOMEBASE_PASSPHRASE_HASH
`

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if err := run(os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr, log); err != nil {
		log.Error("exit", "err", err)
		os.Exit(1)
	}
}

func run(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer, log *slog.Logger) error {
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
	case "hash-passphrase":
		return hashPassphrase(stdin, stdout, stderr)
	case "help", "-h", "--help":
		_, _ = fmt.Fprint(stderr, usage)
		return nil
	default:
		_, _ = fmt.Fprint(stderr, usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func serve(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	if cfg.PassphraseHash == "" {
		return errors.New("HOMEBASE_PASSPHRASE_HASH is not set; create it with: homebase hash-passphrase")
	}
	hash, err := auth.ParseHash(cfg.PassphraseHash)
	if err != nil {
		return fmt.Errorf("HOMEBASE_PASSPHRASE_HASH: %w", err)
	}

	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return fmt.Errorf("create data folder: %w", err)
	}
	database, err := db.Open(filepath.Join(cfg.DataDir, "homebase.db"))
	if err != nil {
		return err
	}
	defer database.Close()
	if err := migrate.Run(ctx, database, migrations.FS, time.Now); err != nil {
		return err
	}

	st := store.NewSQLite(database)
	sy := syncer.New(st, time.Now)
	if err := sy.Init(ctx); err != nil {
		return fmt.Errorf("first-run setup: %w", err)
	}
	if _, err := push.LoadOrCreateKeys(cfg.DataDir); err != nil {
		return err
	}
	blobs, err := files.New(cfg.DataDir)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: api.New(api.Deps{
			Log:   log,
			Auth:  auth.New(st, hash, time.Now),
			Sync:  sy,
			Blobs: blobs,
			Web:   webui.Handler(),
		}),
		ReadHeaderTimeout: 10 * time.Second,
		// Generous enough for a 25 MB upload on a slow phone connection.
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  2 * time.Minute,
		ErrorLog:     slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}

	jobs, stopJobs := context.WithCancel(ctx)
	defer stopJobs()
	go sched.Run(jobs, log, sy, sy, blobs, time.Minute)

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

// hashPassphrase reads the passphrase twice and prints a line ready for an
// environment file. On a terminal, echo is turned off with stty, so no extra
// module is needed.
func hashPassphrase(stdin io.Reader, stdout, stderr io.Writer) error {
	restore := noEcho(stdin)
	defer restore()

	in := bufio.NewReader(stdin)
	read := func(prompt string) (string, error) {
		_, _ = fmt.Fprint(stderr, prompt)
		line, err := in.ReadString('\n')
		_, _ = fmt.Fprintln(stderr)
		if err != nil && !(errors.Is(err, io.EOF) && line != "") {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}

	first, err := read("Passphrase: ")
	if err != nil {
		return err
	}
	second, err := read("Again: ")
	if err != nil {
		return err
	}
	if first != second {
		return errors.New("the two passphrases differ")
	}
	phc, err := auth.HashPassphrase(first)
	if err != nil {
		return err
	}
	// Single quotes keep the $ signs literal in a shell or .env file.
	_, err = fmt.Fprintf(stdout, "HOMEBASE_PASSPHRASE_HASH='%s'\n", phc)
	return err
}

func noEcho(stdin io.Reader) func() {
	f, ok := stdin.(*os.File)
	if !ok {
		return func() {}
	}
	if info, err := f.Stat(); err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return func() {}
	}
	stty := func(arg string) error {
		cmd := exec.Command("stty", arg)
		cmd.Stdin = f
		return cmd.Run()
	}
	if err := stty("-echo"); err != nil {
		return func() {}
	}
	return func() { _ = stty("echo") }
}
