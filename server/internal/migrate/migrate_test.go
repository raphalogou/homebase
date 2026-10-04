package migrate

import (
	"context"
	"testing"
	"testing/fstest"
	"time"

	"homebase/internal/db"
	"homebase/migrations"
)

func TestRunRealMigrations(t *testing.T) {
	d, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ctx := context.Background()
	now := func() time.Time { return time.UnixMilli(1000) }

	if err := Run(ctx, d, migrations.FS, now); err != nil {
		t.Fatalf("first run: %v", err)
	}
	// A second run is a no-op rather than a "table exists" error.
	if err := Run(ctx, d, migrations.FS, now); err != nil {
		t.Fatalf("second run: %v", err)
	}

	var rev int
	if err := d.QueryRow(`SELECT rev FROM meta WHERE id = 1`).Scan(&rev); err != nil {
		t.Fatalf("meta row: %v", err)
	}
	var reminders int
	if err := d.QueryRow(`SELECT count(*) FROM reminders`).Scan(&reminders); err != nil || reminders != 3 {
		t.Fatalf("reminders = %d, %v; want 3", reminders, err)
	}
	var fk int
	if err := d.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil || fk != 1 {
		t.Fatalf("foreign_keys = %d, %v; want 1", fk, err)
	}
}

func TestRunAppliesOnlyNewFiles(t *testing.T) {
	d, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	ctx := context.Background()
	now := func() time.Time { return time.UnixMilli(1000) }

	files := fstest.MapFS{
		"0001_init.sql": {Data: []byte(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL);
CREATE TABLE a (x INTEGER);`)},
	}
	if err := Run(ctx, d, files, now); err != nil {
		t.Fatal(err)
	}
	files["0002_more.sql"] = &fstest.MapFile{Data: []byte(`CREATE TABLE b (y INTEGER);`)}
	if err := Run(ctx, d, files, now); err != nil {
		t.Fatal(err)
	}

	var versions int
	if err := d.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&versions); err != nil || versions != 2 {
		t.Fatalf("schema_migrations rows = %d, %v; want 2", versions, err)
	}
}

func TestRunRejectsBadNames(t *testing.T) {
	tests := []struct {
		name  string
		files fstest.MapFS
	}{
		{"no number", fstest.MapFS{"init.sql": {}}},
		{"duplicate version", fstest.MapFS{"0001_a.sql": {}, "1_b.sql": {}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := db.OpenMemory()
			if err != nil {
				t.Fatal(err)
			}
			defer d.Close()
			if err := Run(context.Background(), d, tt.files, time.Now); err == nil {
				t.Fatal("want error")
			}
		})
	}
}

func TestRunRollsBackFailedMigration(t *testing.T) {
	d, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	files := fstest.MapFS{
		"0001_init.sql": {Data: []byte(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL);`)},
		"0002_bad.sql":  {Data: []byte(`CREATE TABLE c (z INTEGER); THIS IS NOT SQL;`)},
	}
	if err := Run(context.Background(), d, files, time.Now); err == nil {
		t.Fatal("want error")
	}
	var n int
	if err := d.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name = 'c'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("table c exists = %d, %v; want rolled back", n, err)
	}
}
