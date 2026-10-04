package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	web := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("web"))
	})
	h := New(log, web)

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantType   string
		wantBody   string
	}{
		{"healthz", http.MethodGet, "/healthz", http.StatusOK, "text/plain; charset=utf-8", "ok"},
		{"unknown api route", http.MethodGet, "/api/nope", http.StatusNotFound, "application/json", `{"error":{"code":"not_found","message":"No such endpoint."}}`},
		{"web app", http.MethodGet, "/plan", http.StatusOK, "", "web"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantType != "" {
				if got := rec.Header().Get("Content-Type"); got != tt.wantType {
					t.Errorf("Content-Type = %q, want %q", got, tt.wantType)
				}
			}
			if tt.wantBody != "" {
				if got := strings.TrimSpace(rec.Body.String()); got != tt.wantBody {
					t.Errorf("body = %q, want %q", got, tt.wantBody)
				}
			}
		})
	}
}

func TestLoggedPath(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/api/sync", "/api/sync"},
		{"/calendar/s3cret.ics", "/calendar/[redacted]"},
	}
	for _, tt := range tests {
		if got := loggedPath(tt.in); got != tt.want {
			t.Errorf("loggedPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
