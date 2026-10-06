// Package tenant keeps one planner per person. Each lives in its own folder,
// data/users/<id>/, with the same database and files a single install had,
// so sync, rules and queries never need to know about other people.
package tenant

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"homebase/internal/apperr"
	"homebase/internal/auth"
	"homebase/internal/db"
	"homebase/internal/files"
	"homebase/internal/migrate"
	"homebase/internal/store"
	"homebase/internal/syncer"
	"homebase/migrations"
)

// Tenant is one person's planner.
type Tenant struct {
	ID    string
	Dir   string
	Sync  *syncer.Syncer
	Auth  *auth.Auth
	Blobs *files.Blobs
	// Handler serves this person's API; Start sets it.
	Handler http.Handler

	db   *sql.DB
	stop context.CancelFunc
}

// Start runs when a planner opens: it sets Handler and starts the
// background jobs, which run until ctx ends. It gets the registry too,
// since planners that already exist open before Open returns it.
type Start func(ctx context.Context, r *Registry, t *Tenant) error

// Registry holds every planner, open for as long as the server runs, since
// reminders and backups must run for people who are not logged in.
type Registry struct {
	dataDir string
	limiter *auth.Limiter
	now     func() time.Time
	start   Start
	ctx     context.Context

	mu   sync.Mutex
	byID map[string]*Tenant
	// creating serialises Create without holding mu, which the username
	// check inside fill needs.
	creating sync.Mutex
}

// UsersDir is where the planners live, under the data folder.
const UsersDir = "users"

// Open opens every planner under dataDir, after moving an install from
// before several people into the owner's folder.
func Open(ctx context.Context, dataDir string, now func() time.Time, start Start) (*Registry, error) {
	r := &Registry{
		dataDir: dataDir, limiter: auth.NewLimiter(auth.MaxLogins, auth.LoginWindow, now),
		now: now, start: start, ctx: ctx, byID: map[string]*Tenant{},
	}
	if err := r.adoptSingle(); err != nil {
		return nil, fmt.Errorf("move the existing data into %s: %w", UsersDir, err)
	}
	entries, err := os.ReadDir(filepath.Join(dataDir, UsersDir))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		t, err := r.open(e.Name())
		if err != nil {
			r.Close()
			return nil, fmt.Errorf("open planner %s: %w", e.Name(), err)
		}
		r.byID[t.ID] = t
	}
	return r, nil
}

// adoptSingle moves data/homebase.db and data/files, from before several
// people, into data/users/<new id>/. Sessions live in that database, so
// nobody is logged out.
func (r *Registry) adoptSingle() error {
	old := filepath.Join(r.dataDir, "homebase.db")
	if _, err := os.Stat(old); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	dir := filepath.Join(r.dataDir, UsersDir, newID())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	for _, name := range []string{"homebase.db", "homebase.db-wal", "homebase.db-shm", "files"} {
		err := os.Rename(filepath.Join(r.dataDir, name), filepath.Join(dir, name))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (r *Registry) open(id string) (*Tenant, error) {
	dir := filepath.Join(r.dataDir, UsersDir, id)
	d, err := db.Open(filepath.Join(dir, "homebase.db"))
	if err != nil {
		return nil, err
	}
	t, err := r.wrap(id, dir, d)
	if err != nil {
		_ = d.Close()
		return nil, err
	}
	return t, nil
}

func (r *Registry) wrap(id, dir string, d *sql.DB) (*Tenant, error) {
	if err := migrate.Run(r.ctx, d, migrations.FS, r.now); err != nil {
		return nil, err
	}
	st := store.NewSQLite(d)
	sy := syncer.New(st, r.now)
	if err := sy.Init(r.ctx); err != nil {
		return nil, fmt.Errorf("first-run setup: %w", err)
	}
	blobs, err := files.New(dir)
	if err != nil {
		return nil, err
	}
	t := &Tenant{ID: id, Dir: dir, Sync: sy, Auth: auth.NewShared(st, r.limiter, r.now), Blobs: blobs, db: d}
	t.Auth.Taken = func(ctx context.Context, username string) (bool, error) {
		other, err := r.Find(ctx, username)
		return other != nil && other.ID != id, err
	}
	ctx, stop := context.WithCancel(r.ctx)
	t.stop = stop
	if err := r.start(ctx, r, t); err != nil {
		stop()
		return nil, err
	}
	return t, nil
}

// Limiter is the login limit shared by every planner.
func (r *Registry) Limiter() *auth.Limiter { return r.limiter }

// Get returns the planner with this id.
func (r *Registry) Get(id string) (*Tenant, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byID[id]
	return t, ok
}

// All returns every planner.
func (r *Registry) All() []*Tenant {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*Tenant, 0, len(r.byID))
	for _, t := range r.byID {
		out = append(out, t)
	}
	return out
}

// Owner returns the owner's planner. An install from before several people
// has only that one, and its old cookies name no planner.
func (r *Registry) Owner(ctx context.Context) (*Tenant, error) {
	for _, t := range r.All() {
		acc, err := t.Auth.Account(ctx)
		if err == nil && acc.Owner {
			return t, nil
		}
	}
	return nil, nil
}

// Find returns the planner whose username is username, or nil. Usernames
// are lowercase, so the comparison is exact.
// ponytail: opens each account in turn; fine for a household, index it if
// there are ever hundreds.
func (r *Registry) Find(ctx context.Context, username string) (*Tenant, error) {
	u := strings.ToLower(strings.TrimSpace(username))
	for _, t := range r.All() {
		acc, err := t.Auth.Account(ctx)
		if errors.Is(err, store.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if acc.Username == u {
			return t, nil
		}
	}
	return nil, nil
}

// ByCalendarToken returns the planner whose calendar link has token.
func (r *Registry) ByCalendarToken(ctx context.Context, token string) (*Tenant, error) {
	for _, t := range r.All() {
		want, err := t.Sync.CalendarToken(ctx)
		if err != nil {
			return nil, err
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(want)) == 1 {
			return t, nil
		}
	}
	return nil, nil
}

// Count is how many planners exist.
func (r *Registry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.byID)
}

// Create makes a new planner and runs fill on it, which creates its
// account. If fill fails, the planner is removed again. first refuses when
// any planner exists, so two first setups cannot both succeed.
func (r *Registry) Create(first bool, fill func(t *Tenant) error) (*Tenant, error) {
	r.creating.Lock()
	defer r.creating.Unlock()
	if first && r.Count() > 0 {
		return nil, apperr.New(apperr.Invalid, "Homebase is already set up. Log in instead.")
	}
	id := newID()
	dir := filepath.Join(r.dataDir, UsersDir, id)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	// Not yet in byID, so Taken does not see it as someone else.
	t, err := r.open(id)
	if err == nil {
		if err = fill(t); err != nil {
			t.close()
		}
	}
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	r.mu.Lock()
	r.byID[id] = t
	r.mu.Unlock()
	return t, nil
}

// Remove closes a planner and moves its folder to data/removed/, so a
// removal by mistake can be undone by moving it back and restarting.
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byID[id]
	if !ok {
		return apperr.New(apperr.NotFound, "No such person.")
	}
	t.close()
	delete(r.byID, id)
	removed := filepath.Join(r.dataDir, "removed")
	if err := os.MkdirAll(removed, 0o700); err != nil {
		return err
	}
	return os.Rename(t.Dir, filepath.Join(removed, id+"-"+r.now().UTC().Format("20060102-150405")))
}

func (t *Tenant) close() {
	t.stop()
	_ = t.db.Close()
}

// Close closes every planner.
func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.byID {
		t.close()
	}
}

// Person is one entry of the owner's list in Settings.
type Person struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Owner      bool   `json:"owner"`
	MustChange bool   `json:"mustChange"`
}

// People lists everyone, the owner first, then by username.
func (r *Registry) People(ctx context.Context) ([]Person, error) {
	out := []Person{}
	for _, t := range r.All() {
		acc, err := t.Auth.Account(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, Person{ID: t.ID, Username: acc.Username, Owner: acc.Owner, MustChange: acc.MustChange})
	}
	slices.SortFunc(out, func(a, b Person) int {
		if a.Owner != b.Owner {
			if a.Owner {
				return -1
			}
			return 1
		}
		return strings.Compare(a.Username, b.Username)
	})
	return out, nil
}
