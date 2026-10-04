// Command homebase runs the Homebase server.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
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
	"homebase/internal/backup"
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

// Set at build time by the Makefile and Dockerfile (-ldflags -X).
var version, commit = "dev", "unknown"

func main() {
	p := colorFor(os.Stderr, os.Getenv)
	log := newLogger(os.Stderr, p)
	err := run(os.Args[1:], os.Getenv, os.Stdin, os.Stdout, os.Stderr, log)
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "%shomebase: %s%s\n", p.c(red), err, p.c(reset))
	if ue := usageError(""); errors.As(err, &ue) {
		fmt.Fprintln(os.Stderr, "Run 'homebase help' to see the commands and settings.")
		os.Exit(2)
	}
	os.Exit(1)
}

func run(args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer, log *slog.Logger) error {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return usageError("no command given")
	}
	// "homebase serve --help" and the like show the help, not an error.
	for _, a := range args[1:] {
		if a == "-h" || a == "--help" {
			_, err := fmt.Fprint(stdout, usage)
			return err
		}
	}
	switch args[0] {
	case "version", "--version", "-v":
		_, err := fmt.Fprintf(stdout, "homebase %s (%s)\n", version, commit)
		return err
	case "serve":
		if len(args) > 1 {
			return usageError(fmt.Sprintf("serve takes no arguments, got %q; settings come from HOMEBASE_* variables", strings.Join(args[1:], " ")))
		}
		cfg, err := config.Load(getenv)
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		return serve(ctx, cfg, log, stderr, colorFor(stderr, getenv))
	case "hash-passphrase":
		return hashPassphrase(stdin, stdout, stderr)
	case "backup":
		if len(args) != 2 {
			return usageError("backup needs one folder: homebase backup <dir>")
		}
		cfg, err := config.Load(getenv)
		if err != nil {
			return err
		}
		return backup.Run(context.Background(), cfg.DataDir, args[1], time.Now(), stdout)
	case "help", "-h", "--help":
		_, err := fmt.Fprint(stdout, usage)
		return err
	default:
		return usageError(fmt.Sprintf("unknown command %q", args[0]))
	}
}

func serve(ctx context.Context, cfg config.Config, log *slog.Logger, out io.Writer, p palette) error {
	// Checked before anything is created, so a broken variable fails fast.
	if cfg.PassphraseHash != "" {
		if _, err := auth.ParseHash(cfg.PassphraseHash); err != nil {
			return fmt.Errorf("HOMEBASE_PASSPHRASE_HASH: %w", err)
		}
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
	au := auth.New(st, time.Now)
	if cfg.PassphraseHash != "" {
		created, err := au.Bootstrap(ctx, cfg.PassphraseHash)
		if err != nil {
			return fmt.Errorf("account from HOMEBASE_PASSPHRASE_HASH: %w", err)
		}
		if created {
			log.Info("account created from HOMEBASE_PASSPHRASE_HASH", "username", auth.DefaultUsername)
		}
	}
	needsSetup, err := au.SetupNeeded(ctx)
	if err != nil {
		return err
	}
	keys, err := push.LoadOrCreateKeys(cfg.DataDir)
	if err != nil {
		return err
	}
	sender := push.NewSender(keys, cfg.VAPIDSubject, &http.Client{Timeout: 15 * time.Second}, time.Now)
	blobs, err := files.New(cfg.DataDir)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr: cfg.Addr,
		Handler: api.New(api.Deps{
			Log:            log,
			Auth:           au,
			Sync:           sy,
			Blobs:          blobs,
			Keys:           keys,
			Sender:         sender,
			BaseURL:        cfg.BaseURL,
			BackupDir:      cfg.BackupDir,
			BackupInterval: cfg.BackupInterval,
			Web:            webui.Handler(),
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
	reminders := sched.NewReminders(sy, sender, log, time.Now)
	go sched.Run(jobs, log, sy, sy, blobs, reminders, 30*time.Second)
	go backup.Loop(jobs, log, func(ctx context.Context) (bool, error) {
		st, err := sy.GetSettings(ctx)
		return st.Backups != nil && *st.Backups, err
	}, cfg.DataDir, cfg.BackupDir, cfg.BackupInterval)

	// Listening first turns a taken port into a clear error before the
	// summary claims the address.
	ln, err := net.Listen("tcp", cfg.Addr)
	if err != nil {
		return fmt.Errorf("cannot listen on %s (set HOMEBASE_ADDR to another address): %w", cfg.Addr, err)
	}
	settings, err := sy.GetSettings(ctx)
	if err != nil {
		return err
	}
	printSummary(out, p, summary{
		Addr: cfg.Addr, BaseURL: cfg.BaseURL, DataDir: cfg.DataDir,
		BackupDir: cfg.BackupDir, BackupInterval: cfg.BackupInterval,
		Backups: settings.Backups != nil && *settings.Backups, NeedsSetup: needsSetup,
	})
	errc := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.Addr, "version", version, "commit", commit)
		errc <- srv.Serve(ln)
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
	if !isTerminal(f) {
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
