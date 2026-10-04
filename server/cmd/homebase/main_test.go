package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"homebase/internal/auth"
	"homebase/internal/backup"
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

// The variable is optional since accounts live in the database, but a
// broken one still stops the server before it starts.
func TestServeRejectsBadHash(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	env := map[string]string{"HOMEBASE_DATA": t.TempDir(), "HOMEBASE_PASSPHRASE_HASH": "not-a-hash"}
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
	if err := backup.Run(context.Background(), data, dest, now, &out); err != nil {
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
	if err := backup.Run(context.Background(), data, dest, now.Add(24*time.Hour), &out); err != nil {
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

func TestBackupDue(t *testing.T) {
	now := time.Date(2026, 10, 5, 3, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		files []string
		want  bool
	}{
		{"no backups yet", nil, true},
		{"made an hour ago", []string{"homebase-20261005-020000.db"}, false},
		{"a day old, newest wins", []string{"homebase-20261003-030000.db", "homebase-20261004-030030.db"}, true},
		{"newest is recent", []string{"homebase-20261001-030000.db", "homebase-20261004-120000.db"}, false},
		{"name that is not ours", []string{"homebase-copy.db"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, f), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if got := backup.Due(dir, now, 24*time.Hour); got != tt.want {
				t.Errorf("Due = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCLIFriendliness(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	none := func(string) string { return "" }
	var ue usageError
	if err := run([]string{"srve"}, none, nil, io.Discard, io.Discard, log); !errors.As(err, &ue) {
		t.Errorf("unknown command: %v, want a usage error", err)
	}
	var out bytes.Buffer
	if err := run([]string{"backup", "--help"}, none, nil, &out, io.Discard, log); err != nil || !strings.Contains(out.String(), "Commands:") {
		t.Errorf("backup --help: %v %q", err, out.String())
	}

	var b bytes.Buffer
	newLogger(&b, true).With("job", "backup").Error("failed", "err", "disk full")
	if got := b.String(); !strings.Contains(got, "ERROR") || !strings.Contains(got, "failed") ||
		!strings.Contains(got, "job="+reset+"backup") || !strings.Contains(got, red+`"disk full"`) {
		t.Errorf("coloured line = %q", got)
	}

	for addr, want := range map[string]string{":8080": "http://localhost:8080", "127.0.0.1:9000": "http://127.0.0.1:9000", "[::]:80": "http://localhost:80"} {
		if got := localURL(addr); got != want {
			t.Errorf("localURL(%q) = %q, want %q", addr, got, want)
		}
	}
}
