package syncer

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"homebase/internal/store"
)

// Settings is the answer to GET and PUT /api/settings.
type Settings struct {
	TZ        string `json:"tz"`
	WeekStart int    `json:"weekStart"`
}

// GetSettings returns the user's zone and week start.
func (s *Syncer) GetSettings(ctx context.Context) (Settings, error) {
	var out Settings
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		st, err := tx.Settings()
		out = Settings{TZ: st.TZ, WeekStart: st.WeekStart}
		return err
	})
	return out, err
}

// SaveSettings stores a new zone and week start. The zone must be a real
// IANA name; the server embeds the zone database, so any name works.
func (s *Syncer) SaveSettings(ctx context.Context, in Settings) (Settings, error) {
	if in.TZ == "" || len(in.TZ) > 64 {
		return Settings{}, invalid("tz must be an IANA time zone name.")
	}
	if _, err := time.LoadLocation(in.TZ); err != nil || in.TZ == "Local" {
		return Settings{}, invalid("tz must be an IANA time zone name.")
	}
	if in.WeekStart != 0 && in.WeekStart != 1 {
		return Settings{}, invalid("weekStart must be 0 (Sunday) or 1 (Monday).")
	}
	err := s.store.Tx(ctx, func(tx store.Tx) error { return tx.UpdateSettings(in.TZ, in.WeekStart) })
	return in, err
}

// ReminderInput is one slot of PUT /api/reminders.
type ReminderInput struct {
	Slot    int    `json:"slot"`
	Enabled bool   `json:"enabled"`
	AtLocal string `json:"atLocal"`
	Kind    string `json:"kind"`
}

// SaveReminders replaces the given slots. Reminders are synced rows, so
// each write takes a revision like any other.
func (s *Syncer) SaveReminders(ctx context.Context, in []ReminderInput) ([]store.Reminder, error) {
	if len(in) == 0 || len(in) > 3 {
		return nil, invalid("Send one to three reminders.")
	}
	seen := map[int]bool{}
	for _, r := range in {
		if r.Slot < 1 || r.Slot > 3 || seen[r.Slot] {
			return nil, invalid("slot must be 1, 2 or 3, each at most once.")
		}
		seen[r.Slot] = true
		if t, err := time.Parse("15:04", r.AtLocal); err != nil || t.Format("15:04") != r.AtLocal {
			return nil, invalid("atLocal must be a time like 08:00.")
		}
		if !oneOf(r.Kind, "focus", "checkin", "wrap") {
			return nil, invalid("kind must be focus, checkin or wrap.")
		}
	}

	var out []store.Reminder
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		at := s.now().UnixMilli()
		for _, r := range in {
			row := store.Reminder{Slot: r.Slot, Enabled: r.Enabled, AtLocal: r.AtLocal, Kind: r.Kind, UpdatedAt: at}
			var err error
			if row.Rev, err = tx.NextRev(); err != nil {
				return err
			}
			if err := tx.PutReminder(row); err != nil {
				return err
			}
		}
		var err error
		out, err = tx.Reminders()
		return err
	})
	return out, err
}

// ReminderPlan is what the scheduler needs each tick.
func (s *Syncer) ReminderPlan(ctx context.Context) (*time.Location, []store.Reminder, error) {
	var loc *time.Location
	var list []store.Reminder
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		st, err := tx.Settings()
		if err != nil {
			return err
		}
		loc = location(st.TZ)
		list, err = tx.Reminders()
		return err
	})
	return loc, list, err
}

// ClaimReminder records the send before it happens. Only the first claim
// for a slot and day succeeds, so no reminder goes out twice.
func (s *Syncer) ClaimReminder(ctx context.Context, slot int, localDate string) (bool, error) {
	var ok bool
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		var err error
		ok, err = tx.LogReminder(slot, localDate, s.now().UnixMilli())
		return err
	})
	return ok, err
}

// DayState returns the open goals and the tasks planned on localDate.
func (s *Syncer) DayState(ctx context.Context, localDate string) ([]store.Goal, []store.Task, error) {
	var goals []store.Goal
	var tasks []store.Task
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		var err error
		if goals, err = tx.OpenGoals(); err != nil {
			return err
		}
		tasks, err = tx.TasksPlannedOn(localDate)
		return err
	})
	return goals, tasks, err
}

// PushSubs lists every subscription.
func (s *Syncer) PushSubs(ctx context.Context) ([]store.PushSub, error) {
	var list []store.PushSub
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		var err error
		list, err = tx.PushSubs()
		return err
	})
	return list, err
}

// PushDelivered records the outcome of one send: a gone subscription is
// deleted, a delivered one gets its last_ok time.
func (s *Syncer) PushDelivered(ctx context.Context, endpoint string, gone bool) error {
	return s.store.Tx(ctx, func(tx store.Tx) error {
		if gone {
			return tx.DeletePushSub(endpoint)
		}
		return tx.TouchPushSub(endpoint, s.now().UnixMilli())
	})
}

// SubscribeInput is the body of POST /api/push/subscribe.
type SubscribeInput struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256DH string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
	Label string `json:"label"`
}

// Subscribe stores a browser's subscription, replacing one with the same
// endpoint.
func (s *Syncer) Subscribe(ctx context.Context, in SubscribeInput) error {
	u, err := url.Parse(in.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || len(in.Endpoint) > 2048 {
		return invalid("endpoint must be an https URL.")
	}
	if n := decodedLen(in.Keys.P256DH); n != 65 {
		return invalid("keys.p256dh must be a P-256 public key.")
	}
	if n := decodedLen(in.Keys.Auth); n != 16 {
		return invalid("keys.auth must be 16 bytes.")
	}
	label := strings.TrimSpace(strings.ToValidUTF8(in.Label, ""))
	if utf8.RuneCountInString(label) > 100 {
		label = string([]rune(label)[:100])
	}
	return s.store.Tx(ctx, func(tx store.Tx) error {
		return tx.PutPushSub(store.PushSub{
			Endpoint:  in.Endpoint,
			P256DH:    in.Keys.P256DH,
			Auth:      in.Keys.Auth,
			Label:     label,
			CreatedAt: s.now().UnixMilli(),
		})
	})
}

// Device is one subscription as the Reminders screen shows it. The endpoint
// is a bearer secret for the push service, so only its hash leaves the server.
type Device struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	CreatedAt int64  `json:"createdAt"`
	LastOK    *int64 `json:"lastOk"`
}

// DeviceID is the first 16 bytes of the endpoint's SHA-256, in hex. The
// browser can compute it from its own endpoint to find itself in the list.
func DeviceID(endpoint string) string {
	sum := sha256.Sum256([]byte(endpoint))
	return hex.EncodeToString(sum[:16])
}

// Devices lists the subscriptions without their secrets.
func (s *Syncer) Devices(ctx context.Context) ([]Device, error) {
	subs, err := s.PushSubs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Device, len(subs))
	for i, p := range subs {
		out[i] = Device{ID: DeviceID(p.Endpoint), Label: p.Label, CreatedAt: p.CreatedAt, LastOK: p.LastOK}
	}
	return out, nil
}

// Unsubscribe removes a subscription by its endpoint (the device itself) or
// by its id (from another device). Removing one that is not there is fine.
func (s *Syncer) Unsubscribe(ctx context.Context, endpoint, id string) error {
	if endpoint == "" && id == "" {
		return invalid("Send endpoint or id.")
	}
	return s.store.Tx(ctx, func(tx store.Tx) error {
		if endpoint != "" {
			return tx.DeletePushSub(endpoint)
		}
		subs, err := tx.PushSubs()
		if err != nil {
			return err
		}
		for _, p := range subs {
			if DeviceID(p.Endpoint) == id {
				return tx.DeletePushSub(p.Endpoint)
			}
		}
		return nil
	})
}

func decodedLen(s string) int {
	for len(s) > 0 && s[len(s)-1] == '=' {
		s = s[:len(s)-1]
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return -1
	}
	return len(b)
}
