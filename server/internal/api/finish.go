package api

import (
	"net"
	"net/http"
	"strings"
	"time"

	"homebase/internal/apperr"
	"homebase/internal/backup"
	"homebase/internal/ics"
	"homebase/internal/recur"
	"homebase/internal/syncer"
)

func (s *server) handleReview(w http.ResponseWriter, r *http.Request) {
	sum, err := s.Sync.Review(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, sum)
}

func (s *server) handleCompleteReview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WeekStart string            `json:"weekStart"`
		Decisions []syncer.Decision `json:"decisions"`
	}
	if err := decode(w, r, smallBody, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	rev, err := s.Sync.CompleteReview(r.Context(), req.WeekStart, req.Decisions)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, map[string]int64{"rev": rev})
}

// baseURL is HOMEBASE_BASE_URL, or else the address this request came to.
// Behind a local proxy, its X-Forwarded-Proto says whether that was https.
func (s *server) baseURL(r *http.Request) string {
	if s.BaseURL != "" {
		return s.BaseURL
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	} else if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() && r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
	}
	return scheme + "://" + r.Host
}

func (s *server) calendarURL(r *http.Request, token string) string {
	return s.baseURL(r) + "/calendar/" + token + ".ics"
}

func (s *server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.Sync.GetSettings(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	token, err := s.Sync.CalendarToken(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	var lastBackup *int64
	if t, ok := backup.Last(s.BackupDir); ok {
		ms := t.UnixMilli()
		lastBackup = &ms
	}
	writeJSON(w, s.Log, http.StatusOK, map[string]any{
		"tz":           st.TZ,
		"weekStart":    st.WeekStart,
		"calendarUrl":  s.calendarURL(r, token),
		"backups":      st.Backups,
		"backupDir":    s.BackupDir,
		"backupHours":  int(s.BackupInterval.Hours()),
		"lastBackupAt": lastBackup,
	})
}

func (s *server) handleRotateCalendar(w http.ResponseWriter, r *http.Request) {
	token, err := s.Sync.RotateCalendar(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, map[string]string{"url": s.calendarURL(r, token)})
}

// handleCalendar serves the feed to calendar apps, which send no cookie;
// the token in the path is the only key, so a wrong one is a plain 404.
func (s *server) handleCalendar(w http.ResponseWriter, r *http.Request) {
	token, ok := strings.CutSuffix(r.PathValue("file"), ".ics")
	if !ok || token == "" || len(token) > 100 {
		http.NotFound(w, r)
		return
	}
	tasks, tz, found, err := s.Sync.CalendarTasks(r.Context(), token)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	body, etag := ics.Feed(tasks, recur.Date(time.Now().In(loc)))

	h := w.Header()
	h.Set("ETag", etag)
	h.Set("Cache-Control", "private, no-cache")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	h.Set("Content-Type", "text/calendar; charset=utf-8")
	h.Set("Content-Disposition", `inline; filename="homebase.ics"`)
	_, _ = w.Write(body)
}

func (s *server) handleSessions(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie(cookieName)
	token := ""
	if c != nil {
		token = tokenOf(c.Value)
	}
	list, err := s.Auth.Sessions(r.Context(), token)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, list)
}

func (s *server) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decode(w, r, smallBody, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	if req.ID == "" {
		s.writeError(w, r, apperr.New(apperr.Invalid, "Send the id of the session."))
		return
	}
	if err := s.Auth.Revoke(r.Context(), req.ID); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
