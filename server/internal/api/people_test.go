package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func loginAs(t *testing.T, h http.Handler, username, passphrase string) string {
	t.Helper()
	rec := do(h, call{method: "POST", path: "/api/login", body: fmt.Sprintf(`{"username":%q,"passphrase":%q}`, username, passphrase)})
	if rec.Code != http.StatusNoContent {
		t.Fatalf("login %s: %d %s", username, rec.Code, rec.Body)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName {
			return c.Value
		}
	}
	t.Fatal("no session cookie")
	return ""
}

func TestPeople(t *testing.T) {
	h, people := buildPeople(t, http.DefaultClient, true)
	owner := login(t, h)

	add := func(cookie, body string) int {
		return do(h, call{method: "POST", path: "/api/people", body: body, cookie: cookie}).Code
	}
	if code := add(owner, `{"username":"Sam","passphrase":"one time passphrase"}`); code != 200 {
		t.Fatalf("add sam: %d", code)
	}
	for name, body := range map[string]string{
		"a taken username, any case": `{"username":"SAM","passphrase":"one time passphrase"}`,
		"the owner's username":       `{"username":"owner","passphrase":"one time passphrase"}`,
		"a short passphrase":         `{"username":"alex","passphrase":"short"}`,
		"a bad username":             `{"username":"a b","passphrase":"one time passphrase"}`,
	} {
		if code := add(owner, body); code != 400 {
			t.Errorf("%s: %d, want 400", name, code)
		}
	}

	sam := loginAs(t, h, "sam", "one time passphrase")
	var me struct {
		Username   string
		UserID     string
		Owner      bool
		MustChange bool
	}
	_ = json.Unmarshal(do(h, call{method: "GET", path: "/api/me", cookie: sam}).Body.Bytes(), &me)
	if me.Username != "sam" || me.Owner || !me.MustChange || me.UserID == "" {
		t.Fatalf("sam's me = %+v", me)
	}

	// Each person's planner is their own.
	push := fmt.Sprintf(`{"base":0,"ops":[{"op":"upsert","table":"tasks","id":"01HZZZZZZZZZZZZZZZZZZZ0001","updatedAt":%d,"row":{"title":"Owner's task","status":"open"}}]}`, time.Now().UnixMilli())
	if rec := do(h, call{method: "POST", path: "/api/sync", body: push, cookie: owner}); rec.Code != 200 {
		t.Fatalf("owner push: %d %s", rec.Code, rec.Body)
	}
	if body := do(h, call{method: "GET", path: "/api/sync?since=0", cookie: sam}).Body.String(); strings.Contains(body, "Owner's task") {
		t.Fatal("sam can see the owner's task")
	}

	// Only the owner manages people, and nobody takes a used username.
	if code := add(sam, `{"username":"alex","passphrase":"one time passphrase"}`); code != 403 {
		t.Errorf("sam adding someone: %d, want 403", code)
	}
	if code := do(h, call{method: "GET", path: "/api/people", cookie: sam}).Code; code != 403 {
		t.Errorf("sam listing people: %d, want 403", code)
	}
	rename := do(h, call{method: "PUT", path: "/api/account/username", body: `{"username":"owner","passphrase":"one time passphrase"}`, cookie: sam})
	if rename.Code != 400 || !strings.Contains(rename.Body.String(), "taken") {
		t.Errorf("sam taking the owner's name: %d %s", rename.Code, rename.Body)
	}

	// A chosen passphrase ends the one-time one.
	if rec := do(h, call{method: "PUT", path: "/api/account/passphrase", body: `{"current":"one time passphrase","next":"sam's own passphrase"}`, cookie: sam}); rec.Code != 200 {
		t.Fatalf("sam changing passphrase: %d %s", rec.Code, rec.Body)
	}
	_ = json.Unmarshal(do(h, call{method: "GET", path: "/api/me", cookie: sam}).Body.Bytes(), &me)
	if me.MustChange {
		t.Error("mustChange still set after choosing a passphrase")
	}

	var list []struct {
		ID, Username string
		Owner        bool
	}
	_ = json.Unmarshal(do(h, call{method: "GET", path: "/api/people", cookie: owner}).Body.Bytes(), &list)
	if len(list) != 2 || !list[0].Owner || list[1].Username != "sam" {
		t.Fatalf("people = %+v", list)
	}
	if code := do(h, call{method: "POST", path: "/api/people/remove", body: fmt.Sprintf(`{"id":%q}`, list[0].ID), cookie: owner}).Code; code != 400 {
		t.Errorf("removing the owner: %d, want 400", code)
	}
	samDir, _ := people.Get(list[1].ID)
	if code := do(h, call{method: "POST", path: "/api/people/remove", body: fmt.Sprintf(`{"id":%q}`, list[1].ID), cookie: owner}).Code; code != 204 {
		t.Fatalf("removing sam: %d", code)
	}
	if code := do(h, call{method: "GET", path: "/api/me", cookie: sam}).Code; code != 401 {
		t.Errorf("removed sam's session: %d, want 401", code)
	}
	if _, err := os.Stat(samDir.Dir); !os.IsNotExist(err) {
		t.Errorf("sam's folder still in place: %v", err)
	}
	moved, _ := filepath.Glob(filepath.Join(filepath.Dir(filepath.Dir(samDir.Dir)), "removed", list[1].ID+"-*"))
	if len(moved) != 1 {
		t.Errorf("sam's folder not kept in removed/: %v", moved)
	}

	// An unknown username gets the same answer as a wrong passphrase.
	unknown := do(h, call{method: "POST", path: "/api/login", body: `{"username":"nobody","passphrase":"whatever it is"}`})
	wrong := do(h, call{method: "POST", path: "/api/login", body: `{"username":"owner","passphrase":"whatever it is"}`})
	if unknown.Code != wrong.Code || unknown.Body.String() != wrong.Body.String() {
		t.Errorf("unknown %d %s, wrong %d %s", unknown.Code, unknown.Body, wrong.Code, wrong.Body)
	}
}

// A cookie from before several people has no planner id; it is the owner's.
func TestOldCookieIsTheOwners(t *testing.T) {
	h := newServer(t)
	cookie := login(t, h)
	_, token, ok := strings.Cut(cookie, ".")
	if !ok {
		t.Fatalf("cookie %q has no planner id", cookie)
	}
	if code := do(h, call{method: "GET", path: "/api/me", cookie: token}).Code; code != 200 {
		t.Errorf("old-style cookie: %d, want 200", code)
	}
}
