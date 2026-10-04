package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"homebase/internal/apperr"
	"homebase/internal/store"
)

const (
	// SessionLifetime is how long a session lives without being used.
	SessionLifetime = 180 * 24 * time.Hour
	// touchEvery limits writes: last_seen moves at most once an hour.
	touchEvery = time.Hour
	// MaxLogins and LoginWindow: five failures per 10 minutes per address.
	MaxLogins   = 5
	LoginWindow = 10 * time.Minute
)

// Auth checks passphrases and sessions.
type Auth struct {
	store   store.Store
	hash    Hash
	limiter *Limiter
	now     func() time.Time
}

// New returns an Auth. hash comes from HOMEBASE_PASSPHRASE_HASH.
func New(s store.Store, hash Hash, now func() time.Time) *Auth {
	return &Auth{store: s, hash: hash, limiter: NewLimiter(MaxLogins, LoginWindow, now), now: now}
}

// Login checks the passphrase and creates a session. It returns the cookie
// value; only its SHA-256 is stored.
func (a *Auth) Login(ctx context.Context, passphrase, addr, label string) (string, error) {
	if !a.limiter.Allow(addr) {
		return "", apperr.New(apperr.RateLimited, "Too many attempts. Try again in a few minutes.")
	}
	if !a.hash.Matches(passphrase) {
		a.limiter.Fail(addr)
		return "", apperr.New(apperr.Unauthorized, "That passphrase is not right.")
	}
	a.limiter.Reset(addr)

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	now := a.now().UnixMilli()
	err := a.store.Tx(ctx, func(tx store.Tx) error {
		return tx.InsertSession(store.Session{TokenHash: hashToken(token), Label: label, CreatedAt: now, LastSeen: now})
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

// Check validates a session token. refresh is true when the caller should
// send the cookie again to slide its expiry forward.
func (a *Auth) Check(ctx context.Context, token string) (refresh bool, err error) {
	if token == "" || len(token) > 100 {
		return false, apperr.New(apperr.Unauthorized, "Log in to continue.")
	}
	h := hashToken(token)
	now := a.now()
	var found, expired bool
	err = a.store.Tx(ctx, func(tx store.Tx) error {
		s, err := tx.Session(h)
		if errors.Is(err, store.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		idle := now.Sub(time.UnixMilli(s.LastSeen))
		switch {
		case idle > SessionLifetime:
			expired = true
			return tx.DeleteSession(h)
		case idle >= touchEvery:
			refresh = true
			return tx.TouchSession(h, now.UnixMilli())
		}
		return nil
	})
	switch {
	case err != nil:
		return false, err
	case !found:
		return false, apperr.New(apperr.Unauthorized, "Log in to continue.")
	case expired:
		return false, apperr.New(apperr.Unauthorized, "Your session ended. Log in again.")
	}
	return refresh, nil
}

// Logout deletes the session, if any.
func (a *Auth) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return a.store.Tx(ctx, func(tx store.Tx) error { return tx.DeleteSession(hashToken(token)) })
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
