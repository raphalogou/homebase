package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"homebase/internal/auth"
	"homebase/internal/db"
	"homebase/internal/files"
	"homebase/internal/migrate"
	"homebase/internal/push"
	"homebase/internal/store"
	"homebase/internal/syncer"
	"homebase/migrations"
)

const (
	pass = "correct horse battery"
	user = auth.DefaultUsername
)

func newServer(t *testing.T) http.Handler {
	t.Helper()
	return newServerWith(t, http.DefaultClient)
}

// newServerWith uses client to reach push services, so a test can point
// it at a fake one.
func newServerWith(t *testing.T, client *http.Client) http.Handler {
	t.Helper()
	return buildServer(t, client, true)
}

// newFreshServer has no account yet, as on a new install.
func newFreshServer(t *testing.T) http.Handler {
	t.Helper()
	return buildServer(t, http.DefaultClient, false)
}

func buildServer(t *testing.T, client *http.Client, withAccount bool) http.Handler {
	t.Helper()
	d, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	ctx := context.Background()
	if err := migrate.Run(ctx, d, migrations.FS, time.Now); err != nil {
		t.Fatal(err)
	}
	st := store.NewSQLite(d)
	sy := syncer.New(st, time.Now)
	if err := sy.Init(ctx); err != nil {
		t.Fatal(err)
	}
	phc, err := auth.HashPassphrase(pass)
	if err != nil {
		t.Fatal(err)
	}
	au := auth.New(st, time.Now)
	if withAccount {
		if _, err := au.Bootstrap(ctx, phc); err != nil {
			t.Fatal(err)
		}
	}
	web := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("web")) })
	blobs, err := files.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	keys, err := push.LoadOrCreateKeys(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return New(Deps{
		Keys:   keys,
		Sender: push.NewSender(keys, "mailto:me@example.com", client, time.Now),
		Blobs:  blobs,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Auth:   au,
		Sync:   sy,
		Web:    web,
	})
}

type call struct {
	method  string
	path    string
	body    string
	cookie  string
	noCheck bool // leave out X-Homebase and the JSON content type
	addr    string
}

func do(h http.Handler, c call) *httptest.ResponseRecorder {
	var body io.Reader
	if c.body != "" {
		body = strings.NewReader(c.body)
	}
	r := httptest.NewRequest(c.method, c.path, body)
	if !c.noCheck && c.method != http.MethodGet {
		r.Header.Set("X-Homebase", "1")
		r.Header.Set("Content-Type", "application/json")
	}
	if c.cookie != "" {
		r.AddCookie(&http.Cookie{Name: cookieName, Value: c.cookie})
	}
	if c.addr != "" {
		r.RemoteAddr = c.addr
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, r)
	return rec
}

func login(t *testing.T, h http.Handler) string {
	t.Helper()
	rec := do(h, call{method: "POST", path: "/api/login", body: `{"username":"` + user + `","passphrase":"` + pass + `"}`})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("login status = %d: %s", rec.Code, rec.Body)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName {
			if !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.MaxAge != 180*24*3600 {
				t.Errorf("cookie attributes: %+v", c)
			}
			return c.Value
		}
	}
	t.Fatal("no session cookie")
	return ""
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("error body %q: %v", rec.Body, err)
	}
	return body.Error.Code
}

func TestRoutes(t *testing.T) {
	h := newServer(t)
	token := login(t, h)

	tests := []struct {
		name     string
		call     call
		wantCode int
		wantErr  string
	}{
		{"healthz needs no session", call{method: "GET", path: "/healthz"}, 200, ""},
		{"web app needs no session", call{method: "GET", path: "/plan"}, 200, ""},
		{"me without session", call{method: "GET", path: "/api/me"}, 401, "unauthorized"},
		{"me with bad session", call{method: "GET", path: "/api/me", cookie: "nope"}, 401, "unauthorized"},
		{"me", call{method: "GET", path: "/api/me", cookie: token}, 200, ""},
		{"pull", call{method: "GET", path: "/api/sync?since=0&limit=10", cookie: token}, 200, ""},
		{"pull bad since", call{method: "GET", path: "/api/sync?since=x", cookie: token}, 400, "invalid"},
		{"pull limit too high", call{method: "GET", path: "/api/sync?limit=5000", cookie: token}, 400, "invalid"},
		{"push without header", call{method: "POST", path: "/api/sync", body: `{"base":0,"ops":[]}`, cookie: token, noCheck: true}, 400, "invalid"},
		{"push not json", call{method: "POST", path: "/api/sync", body: `base=0`, cookie: token}, 400, "invalid"},
		{"push trailing data", call{method: "POST", path: "/api/sync", body: `{"base":0}{}`, cookie: token}, 400, "invalid"},
		{"push without session", call{method: "POST", path: "/api/sync", body: `{"base":0,"ops":[]}`}, 401, "unauthorized"},
		{"push", call{method: "POST", path: "/api/sync", body: `{"base":0,"ops":[]}`, cookie: token}, 200, ""},
		{"promote unknown task", call{method: "POST", path: "/api/promote", body: `{"taskId":"01HZZZZZZZZZZZZZZZZZZZ0001"}`, cookie: token}, 404, "not_found"},
		{"unknown api route", call{method: "GET", path: "/api/nope", cookie: token}, 404, "not_found"},
		{"login wrong method falls to the API catch-all", call{method: "GET", path: "/api/login"}, 404, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(h, tt.call)
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tt.wantCode, rec.Body)
			}
			if tt.wantErr != "" {
				if got := errCode(t, rec); got != tt.wantErr {
					t.Errorf("error code = %q, want %q", got, tt.wantErr)
				}
			}
		})
	}
}

func TestSyncRoundTrip(t *testing.T) {
	h := newServer(t)
	token := login(t, h)
	now := time.Now().UnixMilli()

	push := `{"base":0,"ops":[
		{"op":"upsert","table":"tasks","id":"01HZZZZZZZZZZZZZZZZZZZ0001","updatedAt":` + itoa(now) + `,"row":{"title":"Easy 5 km run","status":"open"}},
		{"op":"upsert","table":"tasks","id":"01HZZZZZZZZZZZZZZZZZZZ0002","updatedAt":` + itoa(now) + `,"row":{"title":""}}
	]}`
	rec := do(h, call{method: "POST", path: "/api/sync", body: push, cookie: token})
	if rec.Code != 200 {
		t.Fatalf("push: %d %s", rec.Code, rec.Body)
	}
	var res struct {
		Rev      int64
		Applied  []string
		Rejected []struct{ ID, Reason string }
		Changes  struct {
			Tasks []map[string]any
		}
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Applied) != 1 || len(res.Rejected) != 1 || res.Rejected[0].Reason != "invalid" {
		t.Fatalf("push result: %s", rec.Body)
	}
	if len(res.Changes.Tasks) != 1 || res.Changes.Tasks[0]["title"] != "Easy 5 km run" {
		t.Fatalf("changes: %s", rec.Body)
	}
	for _, key := range []string{"plannedOn", "planRank", "projectId", "deletedAt", "doneAt"} {
		if _, ok := res.Changes.Tasks[0][key]; !ok {
			t.Errorf("task JSON lacks %q", key)
		}
	}

	rec = do(h, call{method: "GET", path: "/api/sync?since=0", cookie: token})
	var pull struct {
		Rev       int64
		More      bool
		Tasks     []map[string]any
		Reminders []map[string]any
		Goals     []map[string]any
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &pull); err != nil {
		t.Fatal(err)
	}
	if pull.Rev != res.Rev || pull.More || len(pull.Tasks) != 1 || len(pull.Reminders) != 3 || pull.Goals == nil {
		t.Fatalf("pull: %s", rec.Body)
	}

	rec = do(h, call{method: "GET", path: "/api/me", cookie: token})
	if !strings.Contains(rec.Body.String(), `"tz":"UTC"`) || !strings.Contains(rec.Body.String(), `"weekStart":1`) {
		t.Fatalf("me: %s", rec.Body)
	}
}

func itoa(n int64) string { b, _ := json.Marshal(n); return string(b) }

func TestLoginAndLogout(t *testing.T) {
	h := newServer(t)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{"wrong passphrase", `{"username":"owner","passphrase":"nope"}`, 401},
		{"empty passphrase", `{"username":"owner","passphrase":""}`, 400},
		{"no username", `{"passphrase":"` + pass + `"}`, 400},
		{"wrong username", `{"username":"someone","passphrase":"` + pass + `"}`, 401},
		{"not json", `passphrase`, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(h, call{method: "POST", path: "/api/login", body: tt.body, addr: "203.0.113.9:1234"})
			if rec.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantCode)
			}
		})
	}

	token := login(t, h)
	rec := do(h, call{method: "POST", path: "/api/logout", cookie: token})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("logout = %d", rec.Code)
	}
	if rec := do(h, call{method: "GET", path: "/api/me", cookie: token}); rec.Code != 401 {
		t.Fatalf("me after logout = %d", rec.Code)
	}
}

func TestLoginRateLimitPerAddress(t *testing.T) {
	h := newServer(t)
	wrong := call{method: "POST", path: "/api/login", body: `{"username":"owner","passphrase":"nope"}`, addr: "203.0.113.5:1000"}
	for range auth.MaxLogins {
		if rec := do(h, wrong); rec.Code != 401 {
			t.Fatalf("status = %d", rec.Code)
		}
	}
	rec := do(h, wrong)
	if rec.Code != http.StatusTooManyRequests || errCode(t, rec) != "rate_limited" || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("sixth attempt: %d %s", rec.Code, rec.Body)
	}
	right := call{method: "POST", path: "/api/login", body: `{"username":"` + user + `","passphrase":"` + pass + `"}`, addr: "198.51.100.1:1000"}
	if rec := do(h, right); rec.Code != http.StatusNoContent {
		t.Fatalf("other address: %d", rec.Code)
	}
}

func TestClientAddr(t *testing.T) {
	tests := []struct {
		name   string
		remote string
		xff    []string
		want   string
	}{
		{"direct", "203.0.113.5:4000", nil, "203.0.113.5"},
		{"direct ignores forged header", "203.0.113.5:4000", []string{"10.0.0.1"}, "203.0.113.5"},
		{"behind local proxy", "127.0.0.1:4000", []string{"198.51.100.7"}, "198.51.100.7"},
		{"proxy appends to a forged value", "127.0.0.1:4000", []string{"10.0.0.1, 198.51.100.7"}, "198.51.100.7"},
		{"ipv6 loopback proxy", "[::1]:4000", []string{"2001:db8::1"}, "2001:db8::1"},
		{"garbage header", "127.0.0.1:4000", []string{"not-an-ip"}, "127.0.0.1"},
		{"local without proxy", "127.0.0.1:4000", nil, "127.0.0.1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/login", nil)
			r.RemoteAddr = tt.remote
			for _, v := range tt.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if got := clientAddr(r); got != tt.want {
				t.Errorf("clientAddr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	h := newServer(t)
	for _, path := range []string{"/", "/api/me", "/healthz"} {
		rec := do(h, call{method: "GET", path: path})
		hdr := rec.Header()
		if !strings.Contains(hdr.Get("Content-Security-Policy"), "default-src 'self'") {
			t.Errorf("%s: CSP = %q", path, hdr.Get("Content-Security-Policy"))
		}
		if hdr.Get("X-Content-Type-Options") != "nosniff" || hdr.Get("X-Frame-Options") != "DENY" {
			t.Errorf("%s: headers %v", path, hdr)
		}
	}
	if got := do(h, call{method: "GET", path: "/api/me"}).Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("API Cache-Control = %q", got)
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
