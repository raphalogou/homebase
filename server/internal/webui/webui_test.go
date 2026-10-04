package webui

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestNew(t *testing.T) {
	built := fstest.MapFS{
		"index.html":           {Data: []byte("<!doctype html>app")},
		"assets/app-abc.js":    {Data: []byte("console.log(1)")},
		"manifest.webmanifest": {Data: []byte("{}")},
	}

	tests := []struct {
		name      string
		fs        fstest.MapFS
		method    string
		path      string
		wantCode  int
		wantBody  string
		wantCache string
	}{
		{"root", built, http.MethodGet, "/", http.StatusOK, "<!doctype html>app", "no-cache"},
		{"client route", built, http.MethodGet, "/plan/projects", http.StatusOK, "<!doctype html>app", "no-cache"},
		{"hashed asset", built, http.MethodGet, "/assets/app-abc.js", http.StatusOK, "console.log(1)", "public, max-age=31536000, immutable"},
		{"unhashed file", built, http.MethodGet, "/manifest.webmanifest", http.StatusOK, "{}", "no-cache"},
		{"missing asset", built, http.MethodGet, "/assets/gone.js", http.StatusNotFound, "", ""},
		{"post", built, http.MethodPost, "/", http.StatusMethodNotAllowed, "", ""},
		{"not built", fstest.MapFS{".gitkeep": {}}, http.MethodGet, "/", http.StatusNotFound, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			New(tt.fs).ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
			if got := rec.Header().Get("Cache-Control"); tt.wantCache != "" && got != tt.wantCache {
				t.Errorf("Cache-Control = %q, want %q", got, tt.wantCache)
			}
			if got := rec.Header().Get("X-Content-Type-Options"); rec.Code != http.StatusMethodNotAllowed && got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
			}
		})
	}
}
