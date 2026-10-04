package config

import (
	"path/filepath"
	"testing"
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

	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		{
			name: "defaults",
			env:  map[string]string{},
			want: Config{DataDir: abs("./data"), Addr: ":8080", VAPIDSubject: "mailto:admin@localhost"},
		},
		{
			name: "all set",
			env: map[string]string{
				"HOMEBASE_DATA":          "/srv/homebase",
				"HOMEBASE_ADDR":          "127.0.0.1:9000",
				"HOMEBASE_BASE_URL":      "https://home.example.com/",
				"HOMEBASE_VAPID_SUBJECT": "mailto:me@example.com",
			},
			want: Config{
				DataDir:      "/srv/homebase",
				Addr:         "127.0.0.1:9000",
				BaseURL:      "https://home.example.com",
				VAPIDSubject: "mailto:me@example.com",
			},
		},
		{
			name: "blank values fall back to defaults",
			env:  map[string]string{"HOMEBASE_ADDR": "  "},
			want: Config{DataDir: abs("./data"), Addr: ":8080", VAPIDSubject: "mailto:admin@localhost"},
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
