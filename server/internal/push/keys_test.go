package push

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadOrCreateKeys(t *testing.T) {
	dir := t.TempDir()
	first, err := LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("key file mode = %o, want 600", perm)
	}

	second, err := LoadOrCreateKeys(dir)
	if err != nil {
		t.Fatal(err)
	}
	a, err := first.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	b, err := second.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Error("second load produced a different key")
	}
	raw, err := base64.RawURLEncoding.DecodeString(a)
	if err != nil || len(raw) != 65 || raw[0] != 4 {
		t.Errorf("public key is not an uncompressed P-256 point: %d bytes, %v", len(raw), err)
	}
}

func TestLoadRejectsGarbage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, KeyFile), []byte("not a key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreateKeys(dir); err == nil {
		t.Fatal("want error")
	}
}
