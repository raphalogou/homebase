package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCalendarFeed(t *testing.T) {
	h := newServer(t)
	token := login(t, h)
	today := time.Now().UTC().Format("2006-01-02")
	now := time.Now().UnixMilli()
	push := fmt.Sprintf(`{"base":0,"ops":[{"op":"upsert","table":"tasks","id":"01HZZZZZZZZZZZZZZZZZZZ0001","updatedAt":%d,
		"row":{"title":"Call the bank, about fees","status":"open","due":"%s"}}]}`, now, today)
	if rec := do(h, call{method: "POST", path: "/api/sync", body: push, cookie: token}); rec.Code != 200 {
		t.Fatalf("seed: %d", rec.Code)
	}

	rec := do(h, call{method: "GET", path: "/api/settings", cookie: token})
	var st struct{ CalendarURL string }
	_ = json.Unmarshal(rec.Body.Bytes(), &st)
	path := st.CalendarURL[strings.Index(st.CalendarURL, "/calendar/"):]
	if !strings.HasPrefix(st.CalendarURL, "http://example.com/calendar/") || !strings.HasSuffix(path, ".ics") {
		t.Fatalf("calendarUrl = %q", st.CalendarURL)
	}

	// No cookie: calendar apps send none.
	feed := do(h, call{method: "GET", path: path})
	if feed.Code != 200 || feed.Header().Get("Content-Type") != "text/calendar; charset=utf-8" {
		t.Fatalf("feed: %d %s", feed.Code, feed.Header().Get("Content-Type"))
	}
	if !strings.Contains(feed.Body.String(), `SUMMARY:Due: Call the bank\, about fees`) {
		t.Errorf("feed body: %s", feed.Body)
	}

	r := httptest.NewRequest("GET", path, nil)
	r.Header.Set("If-None-Match", feed.Header().Get("ETag"))
	cached := httptest.NewRecorder()
	h.ServeHTTP(cached, r)
	if cached.Code != http.StatusNotModified {
		t.Errorf("If-None-Match = %d, want 304", cached.Code)
	}

	if rec := do(h, call{method: "GET", path: "/calendar/wrong.ics"}); rec.Code != 404 {
		t.Errorf("wrong token = %d, want 404", rec.Code)
	}

	rotate := do(h, call{method: "POST", path: "/api/calendar/rotate", cookie: token})
	var nu struct{ URL string }
	_ = json.Unmarshal(rotate.Body.Bytes(), &nu)
	if nu.URL == "" || nu.URL == st.CalendarURL {
		t.Fatalf("rotate = %s", rotate.Body)
	}
	if rec := do(h, call{method: "GET", path: path}); rec.Code != 404 {
		t.Errorf("old link after rotate = %d, want 404", rec.Code)
	}
	if rec := do(h, call{method: "GET", path: nu.URL[strings.Index(nu.URL, "/calendar/"):]}); rec.Code != 200 {
		t.Errorf("new link = %d, want 200", rec.Code)
	}
}

func TestReviewRoundTrip(t *testing.T) {
	h := newServer(t)
	token := login(t, h)
	old := time.Now().Add(-30 * 24 * time.Hour).UnixMilli()
	push := fmt.Sprintf(`{"base":0,"ops":[
		{"op":"upsert","table":"projects","id":"01HZZZZZZZZZZZZZZZZZZZ0001","updatedAt":%[1]d,"row":{"title":"Shed","status":"open"}},
		{"op":"upsert","table":"tasks","id":"01HZZZZZZZZZZZZZZZZZZZ0002","updatedAt":%[1]d,"row":{"title":"Paint it","status":"open","projectId":"01HZZZZZZZZZZZZZZZZZZZ0001","createdAt":%[1]d}},
		{"op":"upsert","table":"goals","id":"01HZZZZZZZZZZZZZZZZZZZ0003","updatedAt":%[1]d,"row":{"title":"Learn the cello","status":"open"}}
	]}`, old)
	do(h, call{method: "POST", path: "/api/sync", body: push, cookie: token})

	rec := do(h, call{method: "GET", path: "/api/review", cookie: token})
	var sum struct {
		WeekStart string
		Quiet     []struct{ Kind, ID string }
		Completed bool
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &sum); err != nil || len(sum.Quiet) != 2 || sum.Completed {
		t.Fatalf("review = %s", rec.Body)
	}

	body := fmt.Sprintf(`{"weekStart":%q,"decisions":[
		{"kind":"project","id":"01HZZZZZZZZZZZZZZZZZZZ0001","action":"keep"},
		{"kind":"goal","id":"01HZZZZZZZZZZZZZZZZZZZ0003","action":"pause"}]}`, sum.WeekStart)
	if rec := do(h, call{method: "POST", path: "/api/review/complete", body: body, cookie: token}); rec.Code != 200 {
		t.Fatalf("complete: %d %s", rec.Code, rec.Body)
	}
	rec = do(h, call{method: "GET", path: "/api/review", cookie: token})
	if !strings.Contains(rec.Body.String(), `"completed":true`) || !strings.Contains(rec.Body.String(), `"quiet":[]`) {
		t.Errorf("after review = %s", rec.Body)
	}
	rec = do(h, call{method: "GET", path: "/api/sync?since=0", cookie: token})
	if !strings.Contains(rec.Body.String(), `"title":"Learn the cello","notes":"","status":"paused"`) {
		t.Errorf("goal not paused: %s", rec.Body)
	}

	bad := []string{
		`{"weekStart":"last week","decisions":[]}`,
		`{"weekStart":"2026-03-02","decisions":[{"kind":"task","id":"01HZZZZZZZZZZZZZZZZZZZ0002","action":"keep"}]}`,
		`{"weekStart":"2026-03-02","decisions":[{"kind":"goal","id":"01HZZZZZZZZZZZZZZZZZZZ0003","action":"delete"}]}`,
	}
	for _, b := range bad {
		if rec := do(h, call{method: "POST", path: "/api/review/complete", body: b, cookie: token}); rec.Code != 400 {
			t.Errorf("%s = %d, want 400", b, rec.Code)
		}
	}
}

func TestSessionsOverHTTP(t *testing.T) {
	h := newServer(t)
	phone := login(t, h)
	laptop := login(t, h)
	rec := do(h, call{method: "GET", path: "/api/sessions", cookie: phone})
	var list []struct {
		ID      string
		Current bool
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 2 {
		t.Fatalf("sessions = %s", rec.Body)
	}
	var other string
	for _, s := range list {
		if !s.Current {
			other = s.ID
		}
	}
	if rec := do(h, call{method: "POST", path: "/api/sessions/revoke", body: `{"id":"` + other + `"}`, cookie: phone}); rec.Code != 204 {
		t.Fatalf("revoke: %d", rec.Code)
	}
	if rec := do(h, call{method: "GET", path: "/api/me", cookie: laptop}); rec.Code != 401 {
		t.Errorf("revoked session = %d, want 401", rec.Code)
	}
}

func TestBackupSwitch(t *testing.T) {
	h := newServer(t)
	token := login(t, h)
	get := func() (st struct {
		Backups      bool
		LastBackupAt *int64
	}) {
		rec := do(h, call{method: "GET", path: "/api/settings", cookie: token})
		if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
			t.Fatal(err)
		}
		return st
	}
	if st := get(); st.Backups || st.LastBackupAt != nil {
		t.Fatalf("fresh install: %+v, want off and none made", st)
	}
	put := func(body string) {
		if rec := do(h, call{method: "PUT", path: "/api/settings", body: body, cookie: token}); rec.Code != 200 {
			t.Fatalf("PUT %s: %d %s", body, rec.Code, rec.Body)
		}
	}
	put(`{"tz":"UTC","weekStart":1,"backups":true}`)
	if !get().Backups {
		t.Fatal("switch on did not stick")
	}
	// Saving the zone alone leaves the switch as it is.
	put(`{"tz":"Europe/Paris","weekStart":1}`)
	if !get().Backups {
		t.Fatal("a zone change turned backups off")
	}
	put(`{"tz":"Europe/Paris","weekStart":1,"backups":false}`)
	if get().Backups {
		t.Fatal("switch off did not stick")
	}
}
