package sched

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"homebase/internal/push"
	"homebase/internal/store"
)

type fakeData struct {
	loc     *time.Location
	slots   []store.Reminder
	claimed map[string]bool
	subs    []store.PushSub
	goals   []store.Goal
	planned []store.Task
	gone    []string
	ok      []string
}

func (f *fakeData) ReminderPlan(context.Context) (*time.Location, []store.Reminder, error) {
	return f.loc, f.slots, nil
}

func (f *fakeData) ClaimReminder(_ context.Context, slot int, date string) (bool, error) {
	key := date + "/" + string(rune('0'+slot))
	if f.claimed[key] {
		return false, nil
	}
	f.claimed[key] = true
	return true, nil
}

func (f *fakeData) DayState(context.Context, string) ([]store.Goal, []store.Task, error) {
	return f.goals, f.planned, nil
}

func (f *fakeData) PushSubs(context.Context) ([]store.PushSub, error) { return f.subs, nil }

func (f *fakeData) PushDelivered(_ context.Context, endpoint string, gone bool) error {
	if gone {
		f.gone = append(f.gone, endpoint)
	} else {
		f.ok = append(f.ok, endpoint)
	}
	return nil
}

type fakeSender struct {
	sent    []Message
	failFor map[string]error
}

func (f *fakeSender) Send(_ context.Context, sub push.Subscription, payload []byte, _ time.Duration) error {
	if err := f.failFor[sub.Endpoint]; err != nil {
		return err
	}
	var m Message
	_ = json.Unmarshal(payload, &m)
	f.sent = append(f.sent, m)
	return nil
}

func TestRemindersTick(t *testing.T) {
	paris, _ := time.LoadLocation("Europe/Paris")
	slots := []store.Reminder{
		{Slot: 1, Enabled: true, AtLocal: "08:00", Kind: "focus"},
		{Slot: 2, Enabled: false, AtLocal: "13:00", Kind: "checkin"},
		{Slot: 3, Enabled: true, AtLocal: "20:00", Kind: "wrap"},
	}
	tests := []struct {
		name      string
		at        time.Time // local time of the tick
		wantKinds []string
	}{
		{"before any slot", time.Date(2026, 3, 4, 7, 0, 0, 0, paris), nil},
		{"morning slot", time.Date(2026, 3, 4, 8, 0, 30, 0, paris), []string{"Run a half marathon in spring"}},
		{"disabled midday slot stays quiet", time.Date(2026, 3, 4, 13, 5, 0, 0, paris), nil},
		{"evening slot", time.Date(2026, 3, 4, 20, 10, 0, 0, paris), []string{"Close the day"}},
		{"evening slot missed by hours", time.Date(2026, 3, 4, 23, 0, 0, 0, paris), nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &fakeData{
				loc: paris, slots: slots, claimed: map[string]bool{},
				subs:  []store.PushSub{{Endpoint: "https://push.example/a", Label: "Phone"}},
				goals: []store.Goal{{Title: "Run a half marathon in spring"}},
			}
			sender := &fakeSender{}
			now := tt.at
			r := NewReminders(data, sender, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now.UTC() })

			// Two ticks 30 s apart: the second must not send again.
			for range 2 {
				if err := r.Tick(context.Background()); err != nil {
					t.Fatal(err)
				}
				now = now.Add(30 * time.Second)
			}
			var titles []string
			for _, m := range sender.sent {
				titles = append(titles, m.Title)
			}
			if len(titles) != len(tt.wantKinds) || (len(titles) > 0 && titles[0] != tt.wantKinds[0]) {
				t.Errorf("sent %v, want %v", titles, tt.wantKinds)
			}
		})
	}
}

func TestDeliverRecordsOutcomes(t *testing.T) {
	data := &fakeData{subs: []store.PushSub{
		{Endpoint: "https://push.example/ok", Label: "Phone"},
		{Endpoint: "https://push.example/gone", Label: "Old laptop"},
		{Endpoint: "https://push.example/busy", Label: "Tablet"},
	}}
	sender := &fakeSender{failFor: map[string]error{
		"https://push.example/gone": push.ErrGone,
		"https://push.example/busy": io.ErrUnexpectedEOF,
	}}
	sent, failed := Deliver(context.Background(), data, sender, slog.New(slog.NewTextHandler(io.Discard, nil)), TestMessage())
	if sent != 1 || failed != 2 {
		t.Errorf("sent %d failed %d, want 1 and 2", sent, failed)
	}
	if len(data.ok) != 1 || data.ok[0] != "https://push.example/ok" {
		t.Errorf("last_ok recorded for %v", data.ok)
	}
	if len(data.gone) != 1 || data.gone[0] != "https://push.example/gone" {
		t.Errorf("deleted %v, want only the gone one", data.gone)
	}
}
