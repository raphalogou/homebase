package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"homebase/internal/auth"
	"homebase/internal/db"
	"homebase/internal/files"
	"homebase/internal/migrate"
	"homebase/internal/push"
	"homebase/migrations"
)

func TestHashPassphraseCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	in := strings.NewReader("a long passphrase\na long passphrase\n")
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := run([]string{"hash-passphrase"}, func(string) string { return "" }, in, &out, &errOut, log); err != nil {
		t.Fatal(err)
	}
	line := strings.TrimSpace(out.String())
	phc, ok := strings.CutPrefix(line, "HOMEBASE_PASSPHRASE_HASH='")
	if !ok || !strings.HasSuffix(phc, "'") {
		t.Fatalf("output = %q", line)
	}
	h, err := auth.ParseHash(strings.TrimSuffix(phc, "'"))
	if err != nil {
		t.Fatal(err)
	}
	if !h.Matches("a long passphrase") {
		t.Error("printed hash does not match")
	}
}

func TestHashPassphraseMismatch(t *testing.T) {
	in := strings.NewReader("a long passphrase\nanother passphrase\n")
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := run([]string{"hash-passphrase"}, func(string) string { return "" }, in, io.Discard, io.Discard, log); err == nil {
		t.Fatal("want error")
	}
}

func TestServeNeedsHash(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	env := map[string]string{"HOMEBASE_DATA": t.TempDir()}
	err := run([]string{"serve"}, func(k string) string { return env[k] }, nil, io.Discard, io.Discard, log)
	if err == nil || !strings.Contains(err.Error(), "HOMEBASE_PASSPHRASE_HASH") {
		t.Fatalf("err = %v", err)
	}
}

func TestBackup(t *testing.T) {
	data := t.TempDir()
	d, err := db.Open(filepath.Join(data, "homebase.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := migrate.Run(context.Background(), d, migrations.FS, time.Now); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`UPDATE meta SET rev = 42`); err != nil {
		t.Fatal(err)
	}
	_ = d.Close()
	blobs, err := files.New(data)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := blobs.Save(strings.NewReader("a stored file"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := push.LoadOrCreateKeys(data); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "backups")
	var out bytes.Buffer
	now := time.Date(2026, 3, 4, 2, 0, 0, 0, time.UTC)
	if err := backup(context.Background(), data, dest, now, &out); err != nil {
		t.Fatal(err)
	}

	copyDB, err := db.Open(filepath.Join(dest, "homebase-20260304-020000.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer copyDB.Close()
	var rev int
	if err := copyDB.QueryRow(`SELECT rev FROM meta`).Scan(&rev); err != nil || rev != 42 {
		t.Errorf("copied rev = %d, %v; want 42", rev, err)
	}
	if _, err := os.Stat(filepath.Join(dest, "files", saved.SHA[:2], saved.SHA[2:4], saved.SHA)); err != nil {
		t.Errorf("file not copied: %v", err)
	}
	if info, err := os.Stat(filepath.Join(dest, push.KeyFile)); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("push key not copied privately: %v", err)
	}

	// A second run the next night adds a new database copy and no files.
	out.Reset()
	if err := backup(context.Background(), data, dest, now.Add(24*time.Hour), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "0 new files") {
		t.Errorf("second run: %s", out.String())
	}
}

func TestBackupNeedsAFolder(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := run([]string{"backup"}, func(string) string { return "" }, nil, io.Discard, io.Discard, log); err == nil {
		t.Fatal("want error")
	}
}
