package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"homebase/internal/apperr"
	"homebase/internal/db"
	"homebase/internal/migrate"
	"homebase/internal/store"
	"homebase/migrations"
)

const pass = "correct horse battery"

// testHash is computed once; argon2 at full strength is slow on purpose.
func testHash(t *testing.T) Hash {
	t.Helper()
	s, err := HashPassphrase(pass)
	if err != nil {
		t.Fatal(err)
	}
	h, err := ParseHash(s)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestHashRoundTrip(t *testing.T) {
	h := testHash(t)
	if !h.Matches(pass) {
		t.Error("hash does not match its passphrase")
	}
	if h.Matches(pass + "!") {
		t.Error("hash matches a different passphrase")
	}
	if _, err := HashPassphrase("short"); err == nil {
		t.Error("short passphrase accepted")
	}
}

func TestParseHash(t *testing.T) {
	good, err := HashPassphrase(pass)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		in   string
		ok   bool
	}{
		{"generated", good, true},
		{"empty", "", false},
		{"bcrypt", "$2a$10$abcdefghijklmnopqrstuuvwxyz0123456789ABCDEFGHIJKLMNOPQ", false},
		{"argon2i", strings.Replace(good, "argon2id", "argon2i", 1), false},
		{"huge memory", strings.Replace(good, "m=65536", "m=99999999", 1), false},
		{"bad salt", strings.Replace(good, "$", "$!", 4), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseHash(tt.in)
			if (err == nil) != tt.ok {
				t.Errorf("ParseHash() err = %v, want ok=%v", err, tt.ok)
			}
		})
	}
}

func TestLimiter(t *testing.T) {
	now := time.Unix(0, 0)
	l := NewLimiter(5, 10*time.Minute, func() time.Time { return now })

	for i := range 5 {
		if !l.Allow("a") {
			t.Fatalf("attempt %d blocked", i+1)
		}
		l.Fail("a")
	}
	if l.Allow("a") {
		t.Fatal("sixth attempt allowed")
	}
	if !l.Allow("b") {
		t.Fatal("another address blocked")
	}
	now = now.Add(10*time.Minute + time.Second)
	if !l.Allow("a") {
		t.Fatal("still blocked after the window")
	}
	l.Fail("a")
	l.Reset("a")
	if len(l.failures) != 0 {
		t.Fatal("reset kept failures")
	}
}

type fixture struct {
	ctx  context.Context
	auth *Auth
	now  *time.Time
}

func newFixture(t *testing.T, h Hash) fixture {
	t.Helper()
	d, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	ctx := context.Background()
	if err := migrate.Run(ctx, d, migrations.FS, time.Now); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)
	f := fixture{ctx: ctx, now: &now}
	f.auth = New(store.NewSQLite(d), h, func() time.Time { return *f.now })
	return f
}

func code(err error) apperr.Code {
	if e, ok := apperr.As(err); ok {
		return e.Code
	}
	return ""
}

func TestSessions(t *testing.T) {
	f := newFixture(t, testHash(t))

	if _, err := f.auth.Login(f.ctx, "wrong", "1.2.3.4", "test"); code(err) != apperr.Unauthorized {
		t.Fatalf("wrong passphrase: %v", err)
	}
	token, err := f.auth.Login(f.ctx, pass, "1.2.3.4", "test")
	if err != nil {
		t.Fatal(err)
	}

	if refresh, err := f.auth.Check(f.ctx, token); err != nil || refresh {
		t.Fatalf("fresh session: refresh=%v err=%v", refresh, err)
	}
	*f.now = f.now.Add(2 * time.Hour)
	if refresh, err := f.auth.Check(f.ctx, token); err != nil || !refresh {
		t.Fatalf("after 2h: refresh=%v err=%v, want a refresh", refresh, err)
	}
	if _, err := f.auth.Check(f.ctx, token+"x"); code(err) != apperr.Unauthorized {
		t.Fatalf("wrong token: %v", err)
	}

	// Sliding: used within 180 days of the last use, it keeps working.
	*f.now = f.now.Add(170 * 24 * time.Hour)
	if _, err := f.auth.Check(f.ctx, token); err != nil {
		t.Fatalf("after 170 idle days: %v", err)
	}
	*f.now = f.now.Add(181 * 24 * time.Hour)
	if _, err := f.auth.Check(f.ctx, token); code(err) != apperr.Unauthorized {
		t.Fatalf("after 181 idle days: %v", err)
	}
	*f.now = f.now.Add(-181 * 24 * time.Hour)
	if _, err := f.auth.Check(f.ctx, token); code(err) != apperr.Unauthorized {
		t.Fatalf("expired session came back: %v", err)
	}

	token2, err := f.auth.Login(f.ctx, pass, "1.2.3.4", "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := f.auth.Logout(f.ctx, token2); err != nil {
		t.Fatal(err)
	}
	if _, err := f.auth.Check(f.ctx, token2); code(err) != apperr.Unauthorized {
		t.Fatalf("after logout: %v", err)
	}
}

func TestLoginRateLimit(t *testing.T) {
	f := newFixture(t, testHash(t))
	for range MaxLogins {
		if _, err := f.auth.Login(f.ctx, "wrong", "5.6.7.8", ""); code(err) != apperr.Unauthorized {
			t.Fatalf("got %v", err)
		}
	}
	// Even the right passphrase is refused while limited.
	if _, err := f.auth.Login(f.ctx, pass, "5.6.7.8", ""); code(err) != apperr.RateLimited {
		t.Fatalf("got %v, want rate_limited", err)
	}
	if _, err := f.auth.Login(f.ctx, pass, "9.9.9.9", ""); err != nil {
		t.Fatalf("other address: %v", err)
	}
}
