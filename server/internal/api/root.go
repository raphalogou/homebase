package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"homebase/internal/apperr"
	"homebase/internal/auth"
	"homebase/internal/tenant"
)

// RootDeps are what the routes before a session need.
type RootDeps struct {
	Log    *slog.Logger
	People *tenant.Registry
	// Web serves the built web app for every path that is not an API route.
	Web http.Handler
}

type root struct {
	*server // for writeError and writeChecks
	people  *tenant.Registry
}

// New returns the root handler. Setup, login, logout and the calendar feed
// are answered here; every other API request goes to the planner its
// session cookie names.
func New(d RootDeps) http.Handler {
	rt := &root{server: &server{Deps: Deps{Log: d.Log, People: d.People}}, people: d.People}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /api/setup", rt.handleSetupNeeded)
	mux.Handle("POST /api/setup", rt.writeChecks(http.HandlerFunc(rt.handleSetup)))
	mux.Handle("POST /api/login", rt.writeChecks(http.HandlerFunc(rt.handleLogin)))
	mux.Handle("POST /api/logout", rt.writeChecks(http.HandlerFunc(rt.handleLogout)))
	mux.HandleFunc("GET /calendar/{file}", rt.handleCalendar)
	mux.HandleFunc("/api/", rt.dispatch)
	mux.Handle("/", d.Web)
	return securityHeaders(logRequests(d.Log, mux))
}

// The cookie is "<planner id>.<token>", so a request finds its planner
// without asking every database. A cookie from before several people has no
// id; its session is in the owner's planner.
func cookieValue(id, token string) string { return id + "." + token }

func tokenOf(value string) string {
	if _, token, ok := strings.Cut(value, "."); ok {
		return token
	}
	return value
}

func (rt *root) planner(ctx context.Context, r *http.Request) (*tenant.Tenant, error) {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return nil, nil
	}
	if id, _, ok := strings.Cut(c.Value, "."); ok {
		t, _ := rt.people.Get(id)
		return t, nil
	}
	return rt.people.Owner(ctx)
}

func (rt *root) dispatch(w http.ResponseWriter, r *http.Request) {
	t, err := rt.planner(r.Context(), r)
	if err != nil {
		rt.writeError(w, r, err)
		return
	}
	if t == nil {
		// A removed person's cookie, or none: the same answer as an
		// unknown session, so it cannot probe who exists.
		clearCookie(w)
		rt.writeError(w, r, apperr.New(apperr.Unauthorized, "Log in to continue."))
		return
	}
	t.Handler.ServeHTTP(w, r)
}

func (rt *root) handleSetupNeeded(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, rt.Log, http.StatusOK, map[string]bool{"needed": rt.people.Count() == 0})
}

// handleSetup creates the first person, the owner. It works only while
// nobody exists.
func (rt *root) handleSetup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username   string `json:"username"`
		Passphrase string `json:"passphrase"`
	}
	if err := decode(w, r, smallBody, &req); err != nil {
		rt.writeError(w, r, err)
		return
	}
	if len(req.Username) > 200 {
		rt.writeError(w, r, apperr.ForField(apperr.Invalid, "username", "Use 3 to 32 letters, numbers, dots or dashes."))
		return
	}
	if len(req.Passphrase) > maxSecret {
		rt.writeError(w, r, apperr.ForField(apperr.Invalid, "passphrase", "Use at most 1000 characters."))
		return
	}
	var token string
	t, err := rt.people.Create(true, func(t *tenant.Tenant) error {
		var err error
		token, err = t.Auth.Setup(r.Context(), req.Username, req.Passphrase, clientAddr(r), truncate(r.UserAgent(), 200))
		return err
	})
	if err != nil {
		rt.writeError(w, r, err)
		return
	}
	setCookie(w, cookieValue(t.ID, token))
	w.WriteHeader(http.StatusNoContent)
}

func (rt *root) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username   string `json:"username"`
		Passphrase string `json:"passphrase"`
	}
	if err := decode(w, r, smallBody, &req); err != nil {
		rt.writeError(w, r, err)
		return
	}
	switch {
	case strings.TrimSpace(req.Username) == "" || len(req.Username) > 200:
		rt.writeError(w, r, apperr.ForField(apperr.Invalid, "username", "Enter your username."))
		return
	case req.Passphrase == "" || len(req.Passphrase) > maxSecret:
		rt.writeError(w, r, apperr.ForField(apperr.Invalid, "passphrase", "Enter your passphrase."))
		return
	}
	t, err := rt.people.Find(r.Context(), req.Username)
	if err != nil {
		rt.writeError(w, r, err)
		return
	}
	if t == nil {
		rt.writeError(w, r, auth.Reject(rt.people.Limiter(), req.Passphrase, clientAddr(r)))
		return
	}
	token, err := t.Auth.Login(r.Context(), req.Username, req.Passphrase, clientAddr(r), truncate(r.UserAgent(), 200))
	if err != nil {
		rt.writeError(w, r, err)
		return
	}
	setCookie(w, cookieValue(t.ID, token))
	w.WriteHeader(http.StatusNoContent)
}

func (rt *root) handleLogout(w http.ResponseWriter, r *http.Request) {
	t, err := rt.planner(r.Context(), r)
	if err != nil {
		rt.writeError(w, r, err)
		return
	}
	if c, err := r.Cookie(cookieName); err == nil && t != nil {
		if err := t.Auth.Logout(r.Context(), tokenOf(c.Value)); err != nil {
			rt.writeError(w, r, err)
			return
		}
	}
	clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// handleCalendar finds whose feed the token opens; that planner serves it.
func (rt *root) handleCalendar(w http.ResponseWriter, r *http.Request) {
	token, ok := strings.CutSuffix(r.PathValue("file"), ".ics")
	if !ok || token == "" || len(token) > 100 {
		http.NotFound(w, r)
		return
	}
	t, err := rt.people.ByCalendarToken(r.Context(), token)
	if err != nil {
		rt.writeError(w, r, err)
		return
	}
	if t == nil {
		http.NotFound(w, r)
		return
	}
	t.Handler.ServeHTTP(w, r)
}

// owner lets only the owner through.
func (s *server) owner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acc, err := s.Auth.Account(r.Context())
		if err != nil {
			s.writeError(w, r, err)
			return
		}
		if !acc.Owner {
			s.writeError(w, r, apperr.New(apperr.Forbidden, "Only the owner can manage people."))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) handlePeople(w http.ResponseWriter, r *http.Request) {
	list, err := s.People.People(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, list)
}

// handleAddPerson makes an account with a one-time passphrase, which the
// owner passes on and the person replaces when they first log in.
func (s *server) handleAddPerson(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username   string `json:"username"`
		Passphrase string `json:"passphrase"`
	}
	if err := decode(w, r, smallBody, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	if len(req.Username) > 200 {
		s.writeError(w, r, apperr.ForField(apperr.Invalid, "username", "Use 3 to 32 letters, numbers, dots or dashes."))
		return
	}
	if len(req.Passphrase) > maxSecret {
		s.writeError(w, r, apperr.ForField(apperr.Invalid, "passphrase", "Use at most 1000 characters."))
		return
	}
	var info auth.AccountInfo
	t, err := s.People.Create(false, func(t *tenant.Tenant) error {
		var err error
		info, err = t.Auth.Create(r.Context(), req.Username, req.Passphrase)
		return err
	})
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, tenant.Person{ID: t.ID, Username: info.Username, MustChange: true})
}

func (s *server) handleRemovePerson(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := decode(w, r, smallBody, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	if req.ID == s.UserID {
		s.writeError(w, r, apperr.New(apperr.Invalid, "The owner cannot be removed."))
		return
	}
	if err := s.People.Remove(req.ID); err != nil {
		s.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
