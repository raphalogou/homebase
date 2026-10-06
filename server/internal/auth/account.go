package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"homebase/internal/apperr"
	"homebase/internal/store"
)

// DefaultUsername is given to an install that had only a passphrase, so
// nobody is locked out when usernames arrive. It can be changed in Settings.
const DefaultUsername = "owner"

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9._-]{3,32}$`)
	errLimited      = apperr.New(apperr.RateLimited, "Too many attempts. Try again in 10 minutes.")
	errUsername     = apperr.ForField(apperr.Invalid, "username", "Use 3 to 32 letters, numbers, dots or dashes.")
	errWrongPass    = apperr.ForField(apperr.Invalid, "passphrase", "That passphrase is not right.")
)

// NormalizeUsername lowercases a username and checks its form. Usernames are
// compared without case.
func NormalizeUsername(s string) (string, error) {
	u := strings.ToLower(strings.TrimSpace(s))
	if !usernamePattern.MatchString(u) {
		return "", errUsername
	}
	return u, nil
}

// AccountInfo is what Settings shows about the account.
type AccountInfo struct {
	Username            string `json:"username"`
	PassphraseChangedAt int64  `json:"passphraseChangedAt"`
	Owner               bool   `json:"owner"`
	MustChange          bool   `json:"mustChange"`
}

func info(acc store.Account) AccountInfo {
	return AccountInfo{Username: acc.Username, PassphraseChangedAt: acc.ChangedAt, Owner: acc.Owner, MustChange: acc.MustChange}
}

var errTaken = apperr.ForField(apperr.Invalid, "username", "That username is taken.")

func (a *Auth) taken(ctx context.Context, username string) error {
	if a.Taken == nil {
		return nil
	}
	taken, err := a.Taken(ctx, username)
	if err == nil && taken {
		return errTaken
	}
	return err
}

func (a *Auth) account(ctx context.Context) (store.Account, error) {
	var acc store.Account
	err := a.store.Tx(ctx, func(tx store.Tx) error {
		var err error
		acc, err = tx.Account()
		return err
	})
	return acc, err
}

// matches checks passphrase against a stored PHC string.
func matches(phc, passphrase string) (bool, error) {
	h, err := ParseHash(phc)
	if err != nil {
		return false, fmt.Errorf("stored passphrase hash: %w", err)
	}
	return h.Matches(passphrase), nil
}

// Account returns the username and when the passphrase was last set.
func (a *Auth) Account(ctx context.Context) (AccountInfo, error) {
	acc, err := a.account(ctx)
	if err != nil {
		return AccountInfo{}, err
	}
	return info(acc), nil
}

// SetupNeeded reports whether the account still has to be created.
func (a *Auth) SetupNeeded(ctx context.Context) (bool, error) {
	_, err := a.account(ctx)
	if errors.Is(err, store.ErrNotFound) {
		return true, nil
	}
	return false, err
}

// Bootstrap creates the account from HOMEBASE_PASSPHRASE_HASH when there is
// none yet, so an install from before usernames keeps working. It reports
// whether it created one.
func (a *Auth) Bootstrap(ctx context.Context, phc string) (bool, error) {
	if _, err := ParseHash(phc); err != nil {
		return false, err
	}
	created := false
	err := a.store.Tx(ctx, func(tx store.Tx) error {
		_, err := tx.Account()
		if !errors.Is(err, store.ErrNotFound) {
			return err
		}
		created = true
		return tx.InsertAccount(store.Account{Username: DefaultUsername, PassphraseHash: phc, ChangedAt: a.now().UnixMilli(), Owner: true})
	})
	return created, err
}

// Create makes the account of a person the owner adds. The passphrase is
// one-time: they must choose their own before using Homebase.
func (a *Auth) Create(ctx context.Context, username, passphrase string) (AccountInfo, error) {
	u, err := NormalizeUsername(username)
	if err != nil {
		return AccountInfo{}, err
	}
	if err := CheckNewPassphrase(passphrase); err != nil {
		return AccountInfo{}, apperr.ForField(apperr.Invalid, "passphrase", err.Error())
	}
	if err := a.taken(ctx, u); err != nil {
		return AccountInfo{}, err
	}
	phc, err := HashPassphrase(passphrase)
	if err != nil {
		return AccountInfo{}, err
	}
	acc := store.Account{Username: u, PassphraseHash: phc, ChangedAt: a.now().UnixMilli(), MustChange: true}
	if err := a.store.Tx(ctx, func(tx store.Tx) error { return tx.InsertAccount(acc) }); err != nil {
		return AccountInfo{}, err
	}
	return info(acc), nil
}

// Setup creates the account and logs this device in. It works only while no
// account exists. It shares the login limit, since it is open to anyone who
// reaches a fresh install.
func (a *Auth) Setup(ctx context.Context, username, passphrase, addr, label string) (string, error) {
	if !a.limiter.Allow(addr) {
		return "", errLimited
	}
	u, err := NormalizeUsername(username)
	if err != nil {
		return "", err
	}
	if err := CheckNewPassphrase(passphrase); err != nil {
		return "", apperr.ForField(apperr.Invalid, "passphrase", err.Error())
	}
	if needed, err := a.SetupNeeded(ctx); err != nil {
		return "", err
	} else if !needed {
		a.limiter.Fail(addr)
		return "", apperr.New(apperr.Invalid, "Homebase is already set up. Log in instead.")
	}
	phc, err := HashPassphrase(passphrase)
	if err != nil {
		return "", err
	}
	return a.newSession(ctx, label, func(tx store.Tx) error {
		// Checked again inside the transaction: two setups can race.
		if _, err := tx.Account(); !errors.Is(err, store.ErrNotFound) {
			if err == nil {
				return apperr.New(apperr.Invalid, "Homebase is already set up. Log in instead.")
			}
			return err
		}
		return tx.InsertAccount(store.Account{Username: u, PassphraseHash: phc, ChangedAt: a.now().UnixMilli(), Owner: true})
	})
}

// confirm checks the current passphrase before a change, counting failures
// against the same limit as logins.
func (a *Auth) confirm(ctx context.Context, passphrase, addr string) (store.Account, error) {
	if !a.limiter.Allow(addr) {
		return store.Account{}, errLimited
	}
	acc, err := a.account(ctx)
	if err != nil {
		return store.Account{}, err
	}
	ok, err := matches(acc.PassphraseHash, passphrase)
	if err != nil {
		return store.Account{}, err
	}
	if !ok {
		a.limiter.Fail(addr)
		return store.Account{}, errWrongPass
	}
	return acc, nil
}

// ChangeUsername sets a new username after checking the passphrase.
func (a *Auth) ChangeUsername(ctx context.Context, username, passphrase, addr string) (AccountInfo, error) {
	u, err := NormalizeUsername(username)
	if err != nil {
		return AccountInfo{}, err
	}
	acc, err := a.confirm(ctx, passphrase, addr)
	if err != nil {
		return AccountInfo{}, err
	}
	if subtle.ConstantTimeCompare([]byte(u), []byte(acc.Username)) != 1 {
		if err := a.taken(ctx, u); err != nil {
			return AccountInfo{}, err
		}
		if err := a.store.Tx(ctx, func(tx store.Tx) error { return tx.SetUsername(u) }); err != nil {
			return AccountInfo{}, err
		}
	}
	acc.Username = u
	return info(acc), nil
}

// ChangePassphrase sets a new passphrase after checking the current one, and
// logs out every session except the one making the change.
func (a *Auth) ChangePassphrase(ctx context.Context, token, current, next, addr string) (AccountInfo, error) {
	if err := CheckNewPassphrase(next); err != nil {
		return AccountInfo{}, apperr.ForField(apperr.Invalid, "next", err.Error())
	}
	acc, err := a.confirm(ctx, current, addr)
	if err != nil {
		if e, ok := apperr.As(err); ok && e == errWrongPass {
			return AccountInfo{}, apperr.ForField(apperr.Invalid, "current", e.Message)
		}
		return AccountInfo{}, err
	}
	phc, err := HashPassphrase(next)
	if err != nil {
		return AccountInfo{}, err
	}
	at := a.now().UnixMilli()
	err = a.store.Tx(ctx, func(tx store.Tx) error {
		if err := tx.SetPassphraseHash(phc, at); err != nil {
			return err
		}
		return tx.DeleteSessionsExcept(hashToken(token))
	})
	if err != nil {
		return AccountInfo{}, err
	}
	acc.ChangedAt, acc.MustChange = at, false
	return info(acc), nil
}
