package main

import (
	"bytes"
	"io"
	"log/slog"
	"strings"
	"testing"

	"homebase/internal/auth"
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
