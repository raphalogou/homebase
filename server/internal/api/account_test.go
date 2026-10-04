package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func sessionCookie(rec interface{ Result() *http.Response }) string {
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName && c.Value != "" {
			return c.Value
		}
	}
	return ""
}

func errField(t *testing.T, body []byte) string {
	t.Helper()
	var b struct {
		Error struct{ Field string } `json:"error"`
	}
	if err := json.Unmarshal(body, &b); err != nil {
		t.Fatalf("error body %q: %v", body, err)
	}
	return b.Error.Field
}

func TestSetupFlow(t *testing.T) {
	h := newFreshServer(t)
	rec := do(h, call{method: "GET", path: "/api/setup"})
	if rec.Code != 200 || rec.Body.String() != "{\"needed\":true}\n" {
		t.Fatalf("setup needed: %d %s", rec.Code, rec.Body)
	}

	tests := []struct {
		name, body, field string
		status            int
	}{
		{"bad username", `{"username":"a","passphrase":"a long enough one"}`, "username", 400},
		{"short passphrase", `{"username":"sam","passphrase":"river lamp"}`, "passphrase", 400},
		{"not json", `x`, "", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(h, call{method: "POST", path: "/api/setup", body: tt.body})
			if rec.Code != tt.status || errField(t, rec.Body.Bytes()) != tt.field {
				t.Errorf("got %d %s", rec.Code, rec.Body)
			}
		})
	}
	if rec := do(h, call{method: "POST", path: "/api/setup", body: `{"username":"sam","passphrase":"a long enough one"}`, noCheck: true}); rec.Code != 400 {
		t.Fatalf("setup without the write checks: %d", rec.Code)
	}

	rec = do(h, call{method: "POST", path: "/api/setup", body: `{"username":"Sam","passphrase":"a long enough one"}`})
	token := sessionCookie(rec)
	if rec.Code != http.StatusNoContent || token == "" {
		t.Fatalf("setup: %d %s", rec.Code, rec.Body)
	}
	rec = do(h, call{method: "GET", path: "/api/me", cookie: token})
	var me struct{ Username string }
	if rec.Code != 200 || json.Unmarshal(rec.Body.Bytes(), &me) != nil || me.Username != "sam" {
		t.Fatalf("me after setup: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, call{method: "GET", path: "/api/setup"}); rec.Body.String() != "{\"needed\":false}\n" {
		t.Fatalf("setup still needed: %s", rec.Body)
	}
	if rec := do(h, call{method: "POST", path: "/api/setup", body: `{"username":"eve","passphrase":"another long one"}`}); rec.Code != 400 {
		t.Fatalf("second setup: %d", rec.Code)
	}
	rec = do(h, call{method: "POST", path: "/api/login", body: `{"username":"SAM","passphrase":"a long enough one"}`})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
}

func TestAccountChanges(t *testing.T) {
	h := newServer(t)
	laptop := login(t, h)
	phone := login(t, h)

	if rec := do(h, call{method: "GET", path: "/api/account"}); rec.Code != 401 {
		t.Fatalf("account without a session: %d", rec.Code)
	}
	rec := do(h, call{method: "GET", path: "/api/account", cookie: laptop})
	var info struct {
		Username            string
		PassphraseChangedAt int64
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &info); err != nil || info.Username != user || info.PassphraseChangedAt == 0 {
		t.Fatalf("account: %s %v", rec.Body, err)
	}

	tests := []struct {
		name, path, body, field string
		status                  int
	}{
		{"username: wrong passphrase", "/api/account/username", `{"username":"robin","passphrase":"nope"}`, "passphrase", 400},
		{"username: invalid", "/api/account/username", `{"username":"r b","passphrase":"` + pass + `"}`, "username", 400},
		{"passphrase: no current", "/api/account/passphrase", `{"current":"","next":"a long enough one"}`, "current", 400},
		{"passphrase: wrong current", "/api/account/passphrase", `{"current":"nope","next":"a long enough one"}`, "current", 400},
		{"passphrase: short next", "/api/account/passphrase", `{"current":"` + pass + `","next":"short"}`, "next", 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(h, call{method: "PUT", path: tt.path, body: tt.body, cookie: laptop, addr: "192.0.2.1:1"})
			if rec.Code != tt.status || errField(t, rec.Body.Bytes()) != tt.field {
				t.Errorf("got %d %s", rec.Code, rec.Body)
			}
		})
	}

	rec = do(h, call{method: "PUT", path: "/api/account/username", body: `{"username":"Robin","passphrase":"` + pass + `"}`, cookie: laptop})
	if rec.Code != 200 || !json.Valid(rec.Body.Bytes()) {
		t.Fatalf("change username: %d %s", rec.Code, rec.Body)
	}
	rec = do(h, call{method: "PUT", path: "/api/account/passphrase", body: `{"current":"` + pass + `","next":"maple window harbour 7"}`, cookie: laptop})
	if rec.Code != 200 {
		t.Fatalf("change passphrase: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, call{method: "GET", path: "/api/me", cookie: laptop}); rec.Code != 200 {
		t.Errorf("this device logged out: %d", rec.Code)
	}
	if rec := do(h, call{method: "GET", path: "/api/me", cookie: phone}); rec.Code != 401 {
		t.Errorf("other device still logged in: %d", rec.Code)
	}
	rec = do(h, call{method: "POST", path: "/api/login", body: `{"username":"robin","passphrase":"maple window harbour 7"}`})
	if rec.Code != http.StatusNoContent {
		t.Errorf("login with new details: %d %s", rec.Code, rec.Body)
	}
}
