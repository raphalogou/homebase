package tenant

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"homebase/internal/db"
	"homebase/internal/migrate"
	"homebase/migrations"
)

func noJobs(context.Context, *Registry, *Tenant) error { return nil }

// An install from before several people moves into data/users/<id>/ and
// its account becomes the owner, with its sessions kept.
func TestAdoptSingle(t *testing.T) {
	ctx := context.Background()
	data := t.TempDir()
	d, err := db.Open(filepath.Join(data, "homebase.db"))
	if err != nil {
		t.Fatal(err)
	}
	// The old layout: migrations up to 0003, as it shipped.
	shipped := fstest.MapFS{}
	for _, name := range []string{"0001_init.sql", "0002_account.sql", "0003_backups.sql"} {
		b, err := fs.ReadFile(migrations.FS, name)
		if err != nil {
			t.Fatal(err)
		}
		shipped[name] = &fstest.MapFile{Data: b}
	}
	if err := migrate.Run(ctx, d, shipped, time.Now); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Exec(`INSERT INTO account(id, username, passphrase_hash, changed_at) VALUES (1, 'raphael', 'x', 1)`); err != nil {
		t.Fatal(err)
	}
	_ = d.Close()
	if err := os.MkdirAll(filepath.Join(data, "files", "ab"), 0o700); err != nil {
		t.Fatal(err)
	}

	// A planner that exists already opens inside Open, and its start must
	// still be handed the registry (the owner's People tab needs it).
	var got *Registry
	r, err := Open(ctx, data, time.Now, func(_ context.Context, reg *Registry, _ *Tenant) error {
		got = reg
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if got != r {
		t.Errorf("start got registry %p, want %p", got, r)
	}
	if r.Count() != 1 {
		t.Fatalf("planners = %d, want 1", r.Count())
	}
	if _, err := os.Stat(filepath.Join(data, "homebase.db")); !os.IsNotExist(err) {
		t.Errorf("old database still at the top: %v", err)
	}
	owner, err := r.Owner(ctx)
	if err != nil || owner == nil {
		t.Fatalf("owner = %v, %v", owner, err)
	}
	if _, err := os.Stat(filepath.Join(owner.Dir, "files", "ab")); err != nil {
		t.Errorf("files not moved: %v", err)
	}
	if found, _ := r.Find(ctx, "RAPHAEL"); found != owner {
		t.Error("Find does not find the owner by username")
	}
	if found, _ := r.Find(ctx, "nobody"); found != nil {
		t.Error("Find found someone who does not exist")
	}
}

func TestCreateFirstOnlyOnce(t *testing.T) {
	r, err := Open(context.Background(), t.TempDir(), time.Now, noJobs)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	setup := func(tn *Tenant) error {
		_, err := tn.Auth.Setup(context.Background(), "first", "correct horse battery", "1.2.3.4", "test")
		return err
	}
	if _, err := r.Create(true, setup); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Create(true, setup); err == nil {
		t.Fatal("a second first setup succeeded")
	}
	// A failed fill leaves nothing behind.
	if _, err := r.Create(false, func(tn *Tenant) error {
		_, err := tn.Auth.Create(context.Background(), "first", "another passphrase")
		return err
	}); err == nil {
		t.Fatal("a taken username was created")
	}
	entries, _ := os.ReadDir(filepath.Join(r.dataDir, UsersDir))
	if r.Count() != 1 || len(entries) != 1 {
		t.Errorf("after a failed create: %d planners, %d folders", r.Count(), len(entries))
	}
}
