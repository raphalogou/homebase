package auth

import (
	"testing"
	"time"

	"homebase/internal/apperr"
)

func field(err error) string {
	if e, ok := apperr.As(err); ok {
		return e.Field
	}
	return ""
}

func TestNormalizeUsername(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"sam", "sam", true},
		{"  Sam.Lee_2-x ", "sam.lee_2-x", true},
		{"ab", "", false},
		{"a234567890123456789012345678901b", "a234567890123456789012345678901b", true},
		{"a2345678901234567890123456789012c", "", false},
		{"sam lee", "", false},
		{"sam@home", "", false},
		{"sàm", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := NormalizeUsername(tt.in)
			if (err == nil) != tt.ok || got != tt.want {
				t.Errorf("NormalizeUsername(%q) = %q, %v; want %q ok=%v", tt.in, got, err, tt.want, tt.ok)
			}
			if err != nil && field(err) != "username" {
				t.Errorf("error field = %q", field(err))
			}
		})
	}
}

func TestCheckNewPassphrase(t *testing.T) {
	tests := []struct {
		name, in, msg string
	}{
		{"ten", "river lamp", "Use at least 12 characters. You have 10."},
		{"twelve", "river lamp o", ""},
		{"counts characters, not bytes", "été été été", "Use at least 12 characters. You have 11."},
		{"no composition rules", "aaaaaaaaaaaa", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckNewPassphrase(tt.in)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tt.msg {
				t.Errorf("got %q, want %q", got, tt.msg)
			}
		})
	}
}

func TestSetupOnlyOnce(t *testing.T) {
	f := emptyFixture(t)
	if needed, err := f.auth.SetupNeeded(f.ctx); err != nil || !needed {
		t.Fatalf("fresh install: needed=%v err=%v", needed, err)
	}
	if _, err := f.auth.Login(f.ctx, user, pass, "a", ""); code(err) != apperr.Unauthorized {
		t.Fatalf("login before setup: %v", err)
	}

	tests := []struct {
		name, user, pass, field string
	}{
		{"short username", "ab", pass, "username"},
		{"short passphrase", "Sam", "river lamp", "passphrase"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := f.auth.Setup(f.ctx, tt.user, tt.pass, "a", ""); code(err) != apperr.Invalid || field(err) != tt.field {
				t.Errorf("err = %v (field %q), want invalid on %s", err, field(err), tt.field)
			}
		})
	}

	token, err := f.auth.Setup(f.ctx, "Sam", pass, "a", "phone")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.auth.Check(f.ctx, token); err != nil {
		t.Fatalf("setup did not log in: %v", err)
	}
	info, err := f.auth.Account(f.ctx)
	if err != nil || info.Username != "sam" || info.PassphraseChangedAt != f.now.UnixMilli() {
		t.Fatalf("account = %+v, %v", info, err)
	}
	if _, err := f.auth.Setup(f.ctx, "eve", "another long passphrase", "b", ""); code(err) != apperr.Invalid {
		t.Fatalf("second setup: %v", err)
	}
	if needed, _ := f.auth.SetupNeeded(f.ctx); needed {
		t.Fatal("still needs setup")
	}
}

func TestLoginUsername(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name, user, pass string
		ok               bool
	}{
		{"exact", user, pass, true},
		{"any case and spaces", " SAM ", pass, true},
		{"wrong username", "max", pass, false},
		{"wrong passphrase", user, pass + "x", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.auth.Login(f.ctx, tt.user, tt.pass, "addr-"+tt.name, "")
			if (err == nil) != tt.ok {
				t.Fatalf("err = %v, want ok=%v", err, tt.ok)
			}
			if err != nil {
				e, _ := apperr.As(err)
				if e.Code != apperr.Unauthorized || e.Message != "That username or passphrase did not match." {
					t.Errorf("error %+v says which part was wrong", e)
				}
			}
		})
	}
}

func TestBootstrap(t *testing.T) {
	f := emptyFixture(t)
	phc, err := HashPassphrase(pass)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.auth.Bootstrap(f.ctx, "not a hash"); err == nil {
		t.Fatal("bad hash accepted")
	}
	if created, err := f.auth.Bootstrap(f.ctx, phc); err != nil || !created {
		t.Fatalf("first: created=%v err=%v", created, err)
	}
	if _, err := f.auth.Login(f.ctx, DefaultUsername, pass, "a", ""); err != nil {
		t.Fatalf("login as %s: %v", DefaultUsername, err)
	}

	// Once the account exists, the variable no longer decides anything: a
	// passphrase changed in Settings must survive a restart.
	other, err := HashPassphrase("a different passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if created, err := f.auth.Bootstrap(f.ctx, other); err != nil || created {
		t.Fatalf("second: created=%v err=%v", created, err)
	}
	if _, err := f.auth.Login(f.ctx, DefaultUsername, pass, "b", ""); err != nil {
		t.Fatalf("bootstrap replaced the passphrase: %v", err)
	}
}

func TestChangeUsername(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name, user, pass string
		field            string
	}{
		{"invalid name", "a b", pass, "username"},
		{"wrong passphrase", "robin", "nope", "passphrase"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := f.auth.ChangeUsername(f.ctx, tt.user, tt.pass, "x"); field(err) != tt.field {
				t.Errorf("err = %v, want field %q", err, tt.field)
			}
		})
	}
	info, err := f.auth.ChangeUsername(f.ctx, "Robin", pass, "x")
	if err != nil || info.Username != "robin" {
		t.Fatalf("change: %+v, %v", info, err)
	}
	if _, err := f.auth.Login(f.ctx, user, pass, "y", ""); err == nil {
		t.Error("old username still logs in")
	}
	if _, err := f.auth.Login(f.ctx, "robin", pass, "y", ""); err != nil {
		t.Errorf("new username: %v", err)
	}
}

func TestChangePassphrase(t *testing.T) {
	f := newFixture(t)
	here, err := f.auth.Login(f.ctx, user, pass, "1", "laptop")
	if err != nil {
		t.Fatal(err)
	}
	there, err := f.auth.Login(f.ctx, user, pass, "2", "phone")
	if err != nil {
		t.Fatal(err)
	}
	const next = "maple window harbour 7"

	tests := []struct {
		name, current, next, field string
	}{
		{"too short", pass, "short one", "next"},
		{"wrong current", "nope", next, "current"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := f.auth.ChangePassphrase(f.ctx, here, tt.current, tt.next, "1"); field(err) != tt.field {
				t.Errorf("err = %v, want field %q", err, tt.field)
			}
		})
	}

	*f.now = f.now.Add(time.Hour)
	info, err := f.auth.ChangePassphrase(f.ctx, here, pass, next, "1")
	if err != nil || info.PassphraseChangedAt != f.now.UnixMilli() {
		t.Fatalf("change: %+v, %v", info, err)
	}
	if _, err := f.auth.Check(f.ctx, here); err != nil {
		t.Errorf("this device was logged out: %v", err)
	}
	if _, err := f.auth.Check(f.ctx, there); code(err) != apperr.Unauthorized {
		t.Errorf("other device still logged in: %v", err)
	}
	if _, err := f.auth.Login(f.ctx, user, pass, "3", ""); err == nil {
		t.Error("old passphrase still works")
	}
	if _, err := f.auth.Login(f.ctx, user, next, "3", ""); err != nil {
		t.Errorf("new passphrase: %v", err)
	}
}

// Wrong passphrases on the change forms count toward the login limit, so
// they cannot be used to guess instead.
func TestSharedLimit(t *testing.T) {
	f := newFixture(t)
	for range MaxLogins - 1 {
		if _, err := f.auth.ChangeUsername(f.ctx, "robin", "nope", "9"); field(err) != "passphrase" {
			t.Fatalf("got %v", err)
		}
	}
	if _, err := f.auth.Login(f.ctx, user, "nope", "9", ""); code(err) != apperr.Unauthorized {
		t.Fatalf("got %v", err)
	}
	if _, err := f.auth.Login(f.ctx, user, pass, "9", ""); code(err) != apperr.RateLimited {
		t.Fatalf("login after five failures: %v", err)
	}
	if _, err := f.auth.ChangePassphrase(f.ctx, "t", pass, "a long new passphrase", "9"); code(err) != apperr.RateLimited {
		t.Fatalf("change after five failures: %v", err)
	}
}
