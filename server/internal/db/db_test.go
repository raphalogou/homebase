package db

import (
	"path/filepath"
	"testing"
)

func TestOpenSetsPragmas(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "with space", "..", "home base.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	var mode string
	if err := d.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil || mode != "wal" {
		t.Errorf("journal_mode = %q, %v; want wal", mode, err)
	}
	var fk int
	if err := d.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != 1 {
		t.Errorf("foreign_keys = %d, %v; want 1", fk, err)
	}
	var timeout int
	if err := d.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil || timeout != 5000 {
		t.Errorf("busy_timeout = %d, %v; want 5000", timeout, err)
	}
}
