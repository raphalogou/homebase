// Package api wires the HTTP routes of the server.
package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"homebase/internal/apperr"
	"homebase/internal/auth"
	"homebase/internal/files"
	"homebase/internal/push"
	"homebase/internal/sched"
	"homebase/internal/syncer"
	"homebase/internal/tenant"
)

// Deps are the services one person's handlers use.
type Deps struct {
	Log  *slog.Logger
	Auth *auth.Auth
	Sync *syncer.Syncer
	// UserID names this person's planner; People is everyone, for the
	// owner's list in Settings.
	UserID string
	People *tenant.Registry
	// Blobs holds uploaded files.
	Blobs *files.Blobs
	// Keys and Sender are for Web Push.
	Keys   push.Keys
	Sender sched.Sender
	// BaseURL is HOMEBASE_BASE_URL; empty means "the address requests come to".
	BaseURL string
	// BackupDir and BackupInterval are shown in Settings.
	BackupDir      string
	BackupInterval time.Duration
}

type server struct {
	Deps
}

// Tenant returns the handler for one person's API: everything behind a
// session, and their calendar feed. New routes requests to it.
func Tenant(d Deps) http.Handler {
	s := &server{Deps: d}
	mux := http.NewServeMux()
	mux.Handle("GET /api/account", s.session(http.HandlerFunc(s.handleAccount)))
	mux.Handle("PUT /api/account/username", s.writeChecks(s.session(http.HandlerFunc(s.handleChangeUsername))))
	mux.Handle("PUT /api/account/passphrase", s.writeChecks(s.session(http.HandlerFunc(s.handleChangePassphrase))))
	mux.Handle("GET /api/me", s.session(http.HandlerFunc(s.handleMe)))
	mux.Handle("GET /api/sync", s.session(http.HandlerFunc(s.handlePull)))
	mux.Handle("POST /api/sync", s.writeChecks(s.session(http.HandlerFunc(s.handlePush))))
	mux.Handle("POST /api/promote", s.writeChecks(s.session(http.HandlerFunc(s.handlePromote))))
	mux.Handle("POST /api/files", s.uploadChecks(s.session(http.HandlerFunc(s.handleUpload))))
	mux.Handle("GET /api/files/{sha}", s.session(http.HandlerFunc(s.handleFile)))
	mux.Handle("GET /api/push/key", s.session(http.HandlerFunc(s.handlePushKey)))
	mux.Handle("GET /api/push/subscriptions", s.session(http.HandlerFunc(s.handleDevices)))
	mux.Handle("POST /api/push/subscribe", s.writeChecks(s.session(http.HandlerFunc(s.handleSubscribe))))
	mux.Handle("POST /api/push/unsubscribe", s.writeChecks(s.session(http.HandlerFunc(s.handleUnsubscribe))))
	mux.Handle("POST /api/push/test", s.writeChecks(s.session(http.HandlerFunc(s.handlePushTest))))
	mux.Handle("PUT /api/reminders", s.writeChecks(s.session(http.HandlerFunc(s.handleSaveReminders))))
	mux.Handle("GET /api/settings", s.session(http.HandlerFunc(s.handleGetSettings)))
	mux.Handle("PUT /api/settings", s.writeChecks(s.session(http.HandlerFunc(s.handlePutSettings))))
	mux.Handle("GET /api/review", s.session(http.HandlerFunc(s.handleReview)))
	mux.Handle("POST /api/review/complete", s.writeChecks(s.session(http.HandlerFunc(s.handleCompleteReview))))
	mux.Handle("POST /api/calendar/rotate", s.writeChecks(s.session(http.HandlerFunc(s.handleRotateCalendar))))
	mux.Handle("GET /api/sessions", s.session(http.HandlerFunc(s.handleSessions)))
	mux.Handle("POST /api/sessions/revoke", s.writeChecks(s.session(http.HandlerFunc(s.handleRevokeSession))))
	mux.Handle("GET /api/people", s.session(s.owner(http.HandlerFunc(s.handlePeople))))
	mux.Handle("POST /api/people", s.writeChecks(s.session(s.owner(http.HandlerFunc(s.handleAddPerson)))))
	mux.Handle("POST /api/people/remove", s.writeChecks(s.session(s.owner(http.HandlerFunc(s.handleRemovePerson)))))
	mux.HandleFunc("GET /calendar/{file}", s.handleCalendar)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.writeError(w, r, apperr.New(apperr.NotFound, "No such endpoint."))
	})
	return mux
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte("ok"))
}

// securityHeaders sets the strict policy of docs/SPEC.md section 7 on every
// response: only same-origin scripts, styles, fonts and connections.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data: blob:; object-src 'none'; "+
			"base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// writeChecks enforces the JSON content type and the X-Homebase header on a
// state-changing request. A cross-site form cannot set either, which is the
// CSRF defence on top of SameSite cookies.
func (s *server) writeChecks(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Homebase") != "1" {
			s.writeError(w, r, apperr.New(apperr.Invalid, "Missing X-Homebase header."))
			return
		}
		ct, _, _ := strings.Cut(r.Header.Get("Content-Type"), ";")
		if !strings.EqualFold(strings.TrimSpace(ct), "application/json") {
			s.writeError(w, r, apperr.New(apperr.Invalid, "Content-Type must be application/json."))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// decode reads a JSON body of at most limit bytes into v.
func decode(w http.ResponseWriter, r *http.Request, limit int64, v any) error {
	body := http.MaxBytesReader(w, r.Body, limit)
	dec := json.NewDecoder(body)
	if err := dec.Decode(v); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return apperr.New(apperr.TooLarge, "Request body is too large.")
		}
		return apperr.New(apperr.Invalid, "Request body is not valid JSON for this endpoint.")
	}
	if dec.More() {
		return apperr.New(apperr.Invalid, "Request body has trailing data.")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, log *slog.Logger, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Warn("write response", "err", err)
	}
}

type errorBody struct {
	Error struct {
		Code    apperr.Code `json:"code"`
		Message string      `json:"message"`
		Field   string      `json:"field,omitempty"`
	} `json:"error"`
}

// writeError writes the error shape from docs/SPEC.md section 1. Errors that
// are not *apperr.Error are logged and hidden behind a generic message.
func (s *server) writeError(w http.ResponseWriter, r *http.Request, err error) {
	e, ok := apperr.As(err)
	if !ok {
		s.Log.Error("request failed", "method", r.Method, "path", loggedPath(r.URL.Path), "err", err)
		e = &apperr.Error{Code: "internal", Message: "Something went wrong on the server."}
	}
	var body errorBody
	body.Error.Code = e.Code
	body.Error.Message = e.Message
	body.Error.Field = e.Field
	if e.Code == apperr.RateLimited {
		w.Header().Set("Retry-After", "600")
	}
	writeJSON(w, s.Log, e.Code.Status(), body)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// logRequests logs method, path, status and duration. The query string is
// left out, and calendar paths are redacted because the path is the secret.
func logRequests(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Info("request",
			"method", r.Method,
			"path", loggedPath(r.URL.Path),
			"status", rec.status,
			"duration", time.Since(start),
		)
	})
}

func loggedPath(p string) string {
	if strings.HasPrefix(p, "/calendar/") {
		return "/calendar/[redacted]"
	}
	return p
}
