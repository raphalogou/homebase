package api

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"homebase/internal/apperr"
	"homebase/internal/auth"
	"homebase/internal/syncer"
)

const (
	cookieName = "hb_session"
	smallBody  = 64 << 10
	syncBody   = 16 << 20
)

// session requires a valid session cookie and slides its expiry.
func (s *server) session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(cookieName)
		if err != nil {
			s.writeError(w, r, apperr.New(apperr.Unauthorized, "Log in to continue."))
			return
		}
		refresh, err := s.Auth.Check(r.Context(), tokenOf(c.Value))
		if err != nil {
			if e, ok := apperr.As(err); ok && e.Code == apperr.Unauthorized {
				clearCookie(w)
			}
			s.writeError(w, r, err)
			return
		}
		if refresh {
			setCookie(w, c.Value)
		}
		next.ServeHTTP(w, r)
	})
}

func setCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(auth.SessionLifetime.Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// clientAddr is the address the login limit counts against. Behind Caddy or
// Tailscale the server sees a loopback peer, so only then is the proxy's
// X-Forwarded-For trusted, and only its last entry, which the proxy wrote.
func clientAddr(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return host
	}
	xff := r.Header.Values("X-Forwarded-For")
	if len(xff) == 0 {
		return host
	}
	parts := strings.Split(xff[len(xff)-1], ",")
	if last := strings.TrimSpace(parts[len(parts)-1]); net.ParseIP(last) != nil {
		return last
	}
	return host
}

func truncate(s string, n int) string {
	if !utf8.ValidString(s) {
		s = strings.ToValidUTF8(s, "")
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// maxSecret bounds passphrases at the boundary, well above the 1000
// characters a new one may have, so argon2 never hashes megabytes.
const maxSecret = 4096

func (s *server) handleAccount(w http.ResponseWriter, r *http.Request) {
	info, err := s.Auth.Account(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, info)
}

func (s *server) handleChangeUsername(w http.ResponseWriter, r *http.Request) {
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
	if req.Passphrase == "" || len(req.Passphrase) > maxSecret {
		s.writeError(w, r, apperr.ForField(apperr.Invalid, "passphrase", "Enter your passphrase."))
		return
	}
	info, err := s.Auth.ChangeUsername(r.Context(), req.Username, req.Passphrase, clientAddr(r))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, info)
}

func (s *server) handleChangePassphrase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}
	if err := decode(w, r, smallBody, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	if req.Current == "" || len(req.Current) > maxSecret {
		s.writeError(w, r, apperr.ForField(apperr.Invalid, "current", "Enter your current passphrase."))
		return
	}
	if len(req.Next) > maxSecret {
		s.writeError(w, r, apperr.ForField(apperr.Invalid, "next", "Use at most 1000 characters."))
		return
	}
	c, _ := r.Cookie(cookieName) // the session middleware has checked it
	info, err := s.Auth.ChangePassphrase(r.Context(), tokenOf(c.Value), req.Current, req.Next, clientAddr(r))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, info)
}

// handleMe adds the account to the settings, so a device can prefill the
// username on Log in after its session ends.
func (s *server) handleMe(w http.ResponseWriter, r *http.Request) {
	me, err := s.Sync.Me(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	acc, err := s.Auth.Account(r.Context())
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	// userId lets a device notice that someone else logged in on it.
	writeJSON(w, s.Log, http.StatusOK, struct {
		syncer.Me
		Username   string `json:"username"`
		UserID     string `json:"userId"`
		Owner      bool   `json:"owner"`
		MustChange bool   `json:"mustChange"`
	}{me, acc.Username, s.UserID, acc.Owner, acc.MustChange})
}

func queryInt(r *http.Request, name string, fallback int64) (int64, error) {
	v := r.URL.Query().Get(name)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, apperr.Newf(apperr.Invalid, "%s must be a whole number.", name)
	}
	return n, nil
}

func (s *server) handlePull(w http.ResponseWriter, r *http.Request) {
	since, err := queryInt(r, "since", 0)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	limit, err := queryInt(r, "limit", 500)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	if limit < 1 || limit > syncer.MaxPullLimit {
		s.writeError(w, r, apperr.Newf(apperr.Invalid, "limit must be between 1 and %d.", syncer.MaxPullLimit))
		return
	}
	res, err := s.Sync.Pull(r.Context(), since, int(limit))
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, res)
}

func (s *server) handlePush(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Base int64       `json:"base"`
		Ops  []syncer.Op `json:"ops"`
	}
	if err := decode(w, r, syncBody, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	res, err := s.Sync.Push(r.Context(), req.Base, req.Ops)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, res)
}

func (s *server) handlePromote(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID string  `json:"taskId"`
		GoalID *string `json:"goalId"`
	}
	if err := decode(w, r, smallBody, &req); err != nil {
		s.writeError(w, r, err)
		return
	}
	res, err := s.Sync.Promote(r.Context(), req.TaskID, req.GoalID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, s.Log, http.StatusOK, res)
}
