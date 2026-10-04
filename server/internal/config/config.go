// Package config reads the server configuration from environment variables.
package config

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// Config is the server configuration, read in one place for every variable.
type Config struct {
	DataDir      string
	Addr         string
	BaseURL      string
	VAPIDSubject string
	// PassphraseHash is checked by the serve command, which is the only one
	// that needs it.
	PassphraseHash string
}

// Load reads the configuration through getenv, which is os.Getenv in
// production and a map lookup in tests.
func Load(getenv func(string) string) (Config, error) {
	c := Config{
		DataDir:      envOr(getenv, "HOMEBASE_DATA", "./data"),
		Addr:         envOr(getenv, "HOMEBASE_ADDR", ":8080"),
		BaseURL:      strings.TrimRight(getenv("HOMEBASE_BASE_URL"), "/"),
		VAPIDSubject: envOr(getenv, "HOMEBASE_VAPID_SUBJECT", "mailto:admin@localhost"),
		// Not trimmed: a stray space is a broken hash and should fail loudly.
		PassphraseHash: getenv("HOMEBASE_PASSPHRASE_HASH"),
	}

	dir, err := filepath.Abs(c.DataDir)
	if err != nil {
		return Config{}, fmt.Errorf("HOMEBASE_DATA: %w", err)
	}
	c.DataDir = dir

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

func envOr(getenv func(string) string, key, fallback string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return fallback
}
