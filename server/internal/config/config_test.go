package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	abs := func(p string) string {
		t.Helper()
		a, err := filepath.Abs(p)
		if err != nil {
			t.Fatal(err)
		}
		return a
	}

	const day = 24 * time.Hour
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: Config{DataDir: abs("./data"), Addr: ":8080", VAPIDSubject: "mailto:admin@localhost", BackupDir: abs("./data/backups"), BackupInterval: day},
		},
		{
			name: "all set",
			env: map[string]string{
				"HOMEBASE_DATA":            "/srv/homebase",
				"HOMEBASE_ADDR":            "127.0.0.1:9000",
				"HOMEBASE_BASE_URL":        "https://home.example.com/",
				"HOMEBASE_VAPID_SUBJECT":   "mailto:me@example.com",
				"HOMEBASE_PASSPHRASE_HASH": "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$a2V5",
			},
			want: Config{
				DataDir:        "/srv/homebase",
				Addr:           "127.0.0.1:9000",
				BaseURL:        "https://home.example.com",
				VAPIDSubject:   "mailto:me@example.com",
				PassphraseHash: "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$a2V5",
				BackupDir:      "/srv/homebase/backups",
				BackupInterval: day,
			},
		},
		{
			name: "backup folder elsewhere",
			env:  map[string]string{"HOMEBASE_DATA": "/srv/homebase", "HOMEBASE_BACKUP_DIR": "/mnt/backups"},
			want: Config{DataDir: "/srv/homebase", Addr: ":8080", VAPIDSubject: "mailto:admin@localhost", BackupDir: "/mnt/backups", BackupInterval: day},
		},
		{
			name: "blank values fall back to defaults",
			env:  map[string]string{"HOMEBASE_ADDR": "  "},
			want: Config{DataDir: abs("./data"), Addr: ":8080", VAPIDSubject: "mailto:admin@localhost", BackupDir: abs("./data/backups"), BackupInterval: day},
		},
		{
			name:    "relative base URL",
			env:     map[string]string{"HOMEBASE_BASE_URL": "home.example.com"},
			wantErr: true,
		},
		{
			name:    "bad VAPID subject",
			env:     map[string]string{"HOMEBASE_VAPID_SUBJECT": "me@example.com"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(func(k string) string { return tt.env[k] })
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() = %+v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPassphraseHashQuotes(t *testing.T) {
	tests := []struct{ in, want string }{
		{"$argon2id$x", "$argon2id$x"},
		{"'$argon2id$x'", "$argon2id$x"},
		{`"$argon2id$x"`, "$argon2id$x"},
		{`'$argon2id$x"`, `'$argon2id$x"`},
		{"'", "'"},
	}
	for _, tt := range tests {
		c, err := Load(func(k string) string {
			if k == "HOMEBASE_PASSPHRASE_HASH" {
				return tt.in
			}
			return ""
		})
		if err != nil {
			t.Fatal(err)
		}
		if c.PassphraseHash != tt.want {
			t.Errorf("%q -> %q, want %q", tt.in, c.PassphraseHash, tt.want)
		}
	}
}

func TestBackupInterval(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"", 24 * time.Hour, true},
		{"7d", 7 * 24 * time.Hour, true},
		{"1d", 24 * time.Hour, true},
		{"12h", 12 * time.Hour, true},
		{"90m", 90 * time.Minute, true},
		{"1m", 0, false},
		{"0d", 0, false},
		{"1.5d", 0, false},
		{"-2d", 0, false},
		{"week", 0, false},
	}
	for _, tt := range tests {
		c, err := Load(func(k string) string {
			if k == "HOMEBASE_BACKUP_INTERVAL" {
				return tt.in
			}
			return ""
		})
		if (err == nil) != tt.ok || (tt.ok && c.BackupInterval != tt.want) {
			t.Errorf("%q -> %v, %v; want %v, ok %v", tt.in, c.BackupInterval, err, tt.want, tt.ok)
		}
	}
}
