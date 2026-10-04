package api

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"homebase/internal/syncer"
)

func subscribeBody(endpoint, label string) string {
	k, _ := ecdh.P256().GenerateKey(rand.Reader)
	auth := make([]byte, 16)
	_, _ = rand.Read(auth)
	b, _ := json.Marshal(map[string]any{
		"endpoint": endpoint,
		"keys": map[string]string{
			"p256dh": base64.RawURLEncoding.EncodeToString(k.PublicKey().Bytes()),
			"auth":   base64.RawURLEncoding.EncodeToString(auth),
		},
		"label": label,
	})
	return string(b)
}

func TestSubscribeValidation(t *testing.T) {
	h := newServer(t)
	token := login(t, h)
	good := subscribeBody("https://push.example/a", "Chrome on Android")
	tests := []struct {
		name string
		body string
		want int
	}{
		{"good", good, 204},
		{"same again", good, 204},
		{"http endpoint", subscribeBody("http://push.example/a", "x"), 400},
		{"short key", strings.Replace(good, `"p256dh":"`, `"p256dh":"AAAA`, 1), 400},
		{"bad auth", `{"endpoint":"https://push.example/b","keys":{"p256dh":"x","auth":"y"}}`, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(h, call{method: "POST", path: "/api/push/subscribe", body: tt.body, cookie: token})
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}
}

func TestDevicesListAndRemove(t *testing.T) {
	h := newServer(t)
	token := login(t, h)
	for _, ep := range []string{"https://push.example/phone", "https://push.example/laptop"} {
		if rec := do(h, call{method: "POST", path: "/api/push/subscribe", body: subscribeBody(ep, ep[len(ep)-5:]), cookie: token}); rec.Code != 204 {
			t.Fatalf("subscribe: %d", rec.Code)
		}
	}
	rec := do(h, call{method: "GET", path: "/api/push/subscriptions", cookie: token})
	if strings.Contains(rec.Body.String(), "push.example") {
		t.Fatalf("device list leaks endpoints: %s", rec.Body)
	}
	var list []syncer.Device
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 2 {
		t.Fatalf("devices = %s", rec.Body)
	}
	if list[0].ID != syncer.DeviceID("https://push.example/phone") {
		t.Errorf("id is not the endpoint's hash")
	}

	// Remove the laptop by id, as another device would.
	body, _ := json.Marshal(map[string]string{"id": list[1].ID})
	if rec := do(h, call{method: "POST", path: "/api/push/unsubscribe", body: string(body), cookie: token}); rec.Code != 204 {
		t.Fatalf("unsubscribe by id: %d", rec.Code)
	}
	// And the phone by its endpoint, as the phone itself would.
	if rec := do(h, call{method: "POST", path: "/api/push/unsubscribe", body: `{"endpoint":"https://push.example/phone"}`, cookie: token}); rec.Code != 204 {
		t.Fatalf("unsubscribe by endpoint: %d", rec.Code)
	}
	rec = do(h, call{method: "GET", path: "/api/push/subscriptions", cookie: token})
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Errorf("devices after removal = %s", rec.Body)
	}
	if rec := do(h, call{method: "POST", path: "/api/push/unsubscribe", body: `{}`, cookie: token}); rec.Code != 400 {
		t.Errorf("unsubscribe with nothing = %d, want 400", rec.Code)
	}
}

func TestPushTestReachesAPushService(t *testing.T) {
	var mu sync.Mutex
	got := map[string]int{}
	svc := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		mu.Lock()
		got[r.URL.Path]++
		mu.Unlock()
		if r.URL.Path == "/gone" {
			w.WriteHeader(http.StatusGone)
			return
		}
		if !strings.HasPrefix(r.Header.Get("Authorization"), "vapid t=") || r.Header.Get("Content-Encoding") != "aes128gcm" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer svc.Close()

	h := newServerWith(t, svc.Client())
	token := login(t, h)
	for _, path := range []string{"/phone", "/gone"} {
		do(h, call{method: "POST", path: "/api/push/subscribe", body: subscribeBody(svc.URL+path, path), cookie: token})
	}

	rec := do(h, call{method: "POST", path: "/api/push/test", cookie: token})
	if !strings.Contains(rec.Body.String(), `"sent":1`) || !strings.Contains(rec.Body.String(), `"failed":1`) {
		t.Fatalf("test result = %s", rec.Body)
	}
	if got["/phone"] != 1 || got["/gone"] != 1 {
		t.Errorf("service saw %v", got)
	}
	// The gone subscription was removed, so a second test only sends one.
	rec = do(h, call{method: "POST", path: "/api/push/test", cookie: token})
	if !strings.Contains(rec.Body.String(), `"sent":1`) || !strings.Contains(rec.Body.String(), `"failed":0`) {
		t.Errorf("second test = %s", rec.Body)
	}
	if got["/gone"] != 1 {
		t.Errorf("gone subscription was tried again")
	}

	rec = do(h, call{method: "GET", path: "/api/push/key", cookie: token})
	var key struct{ PublicKey string }
	_ = json.Unmarshal(rec.Body.Bytes(), &key)
	if raw, err := base64.RawURLEncoding.DecodeString(key.PublicKey); err != nil || len(raw) != 65 {
		t.Errorf("public key = %q", key.PublicKey)
	}
}

func TestSaveRemindersAndSettings(t *testing.T) {
	h := newServer(t)
	token := login(t, h)
	tests := []struct {
		name string
		path string
		body string
		want int
	}{
		{"reminders", "/api/reminders", `[{"slot":1,"enabled":true,"atLocal":"07:30","kind":"focus"},{"slot":2,"enabled":true,"atLocal":"12:45","kind":"checkin"}]`, 200},
		{"bad time", "/api/reminders", `[{"slot":1,"enabled":true,"atLocal":"7:30","kind":"focus"}]`, 400},
		{"25:00", "/api/reminders", `[{"slot":1,"enabled":true,"atLocal":"25:00","kind":"focus"}]`, 400},
		{"slot 4", "/api/reminders", `[{"slot":4,"enabled":true,"atLocal":"07:30","kind":"focus"}]`, 400},
		{"same slot twice", "/api/reminders", `[{"slot":1,"enabled":true,"atLocal":"07:30","kind":"focus"},{"slot":1,"enabled":false,"atLocal":"07:30","kind":"focus"}]`, 400},
		{"bad kind", "/api/reminders", `[{"slot":1,"enabled":true,"atLocal":"07:30","kind":"nag"}]`, 400},
		{"zone", "/api/settings", `{"tz":"Europe/Paris","weekStart":1}`, 200},
		{"unknown zone", "/api/settings", `{"tz":"Mars/Olympus","weekStart":1}`, 400},
		{"Local is not a zone", "/api/settings", `{"tz":"Local","weekStart":1}`, 400},
		{"bad week start", "/api/settings", `{"tz":"UTC","weekStart":3}`, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(h, call{method: "PUT", path: tt.path, body: tt.body, cookie: token})
			if rec.Code != tt.want {
				t.Errorf("status = %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}

	// Saved reminders reach other devices through sync, with new revisions.
	rec := do(h, call{method: "GET", path: "/api/sync?since=3", cookie: token})
	if !strings.Contains(rec.Body.String(), `"atLocal":"07:30"`) || !strings.Contains(rec.Body.String(), `"atLocal":"12:45"`) {
		t.Errorf("sync after saving reminders = %s", rec.Body)
	}
	rec = do(h, call{method: "GET", path: "/api/me", cookie: token})
	if !strings.Contains(rec.Body.String(), `"tz":"Europe/Paris"`) {
		t.Errorf("me = %s", rec.Body)
	}
}
