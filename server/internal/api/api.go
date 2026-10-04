// Package api wires the HTTP routes of the server.
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// New returns the root handler. web serves the built web app for every path
// that is not an API route.
func New(log *slog.Logger, web http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, log, http.StatusNotFound, "not_found", "No such endpoint.")
	})
	mux.Handle("/", web)
	return logRequests(log, mux)
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte("ok"))
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// writeError writes the error shape from docs/SPEC.md section 1.
func writeError(w http.ResponseWriter, log *slog.Logger, status int, code, message string) {
	var body errorBody
	body.Error.Code = code
	body.Error.Message = message
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Warn("write error response", "err", err)
	}
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
