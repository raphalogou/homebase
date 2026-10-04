// Package config reads the server configuration from environment variables.
package config

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config is the server configuration, read in one place for every variable.
type Config struct {
	DataDir      string
	Addr         string
	BaseURL      string
	VAPIDSubject string
	// PassphraseHash is optional. An install from before usernames sets it;
	// the server then creates the account from it once (auth.Bootstrap).
	PassphraseHash string
	// BackupDir is where serve puts its backups once they are
	// switched on in Settings: HOMEBASE_BACKUP_DIR, or "backups" in the data
	// folder. Another disk is safer; the default needs no setup.
	BackupDir string
	// BackupInterval is how often those backups are made:
	// HOMEBASE_BACKUP_INTERVAL, a Go duration or whole days ("7d"), 1d by
	// default.
	BackupInterval time.Duration
}

// Load reads the configuration through getenv, which is os.Getenv in
// production and a map lookup in tests.
func Load(getenv func(string) string) (Config, error) {
	c := Config{
		DataDir:        envOr(getenv, "HOMEBASE_DATA", "./data"),
		Addr:           envOr(getenv, "HOMEBASE_ADDR", ":8080"),
		BaseURL:        strings.TrimRight(getenv("HOMEBASE_BASE_URL"), "/"),
		VAPIDSubject:   envOr(getenv, "HOMEBASE_VAPID_SUBJECT", "mailto:admin@localhost"),
		PassphraseHash: unquote(getenv("HOMEBASE_PASSPHRASE_HASH")),
		BackupDir:      strings.TrimSpace(getenv("HOMEBASE_BACKUP_DIR")),
	}

	dir, err := filepath.Abs(c.DataDir)
	if err != nil {
		return Config{}, fmt.Errorf("HOMEBASE_DATA: %w", err)
	}
	c.DataDir = dir

	if c.BackupDir == "" {
		c.BackupDir = filepath.Join(dir, "backups")
	} else if c.BackupDir, err = filepath.Abs(c.BackupDir); err != nil {
		return Config{}, fmt.Errorf("HOMEBASE_BACKUP_DIR: %w", err)
	}

	if c.BackupInterval, err = interval(envOr(getenv, "HOMEBASE_BACKUP_INTERVAL", "1d")); err != nil {
		return Config{}, fmt.Errorf("HOMEBASE_BACKUP_INTERVAL: %w", err)
	}

	if c.BaseURL != "" {
		u, err := url.Parse(c.BaseURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
			return Config{}, fmt.Errorf("HOMEBASE_BASE_URL must be an absolute http or https URL, got %q", c.BaseURL)
		}
	}

	if !strings.HasPrefix(c.VAPIDSubject, "mailto:") && !strings.HasPrefix(c.VAPIDSubject, "https://") {
		return Config{}, fmt.Errorf("HOMEBASE_VAPID_SUBJECT must start with mailto: or https://, got %q", c.VAPIDSubject)
	}

	return c, nil
}

// interval reads a Go duration ("12h") or whole days ("7d"), which Go's
// format lacks. Under an hour is refused: a typo like "1m" would fill the
// disk with copies.
func interval(v string) (time.Duration, error) {
	var d time.Duration
	var err error
	if days, ok := strings.CutSuffix(v, "d"); ok {
		var n int
		n, err = strconv.Atoi(days)
		d = time.Duration(n) * 24 * time.Hour
	} else {
		d, err = time.ParseDuration(v)
	}
	if err != nil || d < time.Hour {
		return 0, fmt.Errorf("want whole days like 7d or a duration like 12h, at least 1h; got %q", v)
	}
	return d, nil
}

// unquote removes one pair of matching quotes. hash-passphrase prints the
// hash in single quotes for shells and systemd, which strip them; Docker's
// --env-file keeps them. Spaces are not trimmed: a stray one is a broken
// hash and should fail loudly.
func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

func envOr(getenv func(string) string, key, fallback string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return fallback
}
