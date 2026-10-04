package sched

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"homebase/internal/push"
	"homebase/internal/recur"
	"homebase/internal/store"
)

// ReminderData is the part of the syncer the reminders use.
type ReminderData interface {
	ReminderPlan(ctx context.Context) (*time.Location, []store.Reminder, error)
	ClaimReminder(ctx context.Context, slot int, localDate string) (bool, error)
	DayState(ctx context.Context, localDate string) ([]store.Goal, []store.Task, error)
	PushSubs(ctx context.Context) ([]store.PushSub, error)
	PushDelivered(ctx context.Context, endpoint string, gone bool) error
}

// Sender sends one encrypted message to one subscription.
type Sender interface {
	Send(ctx context.Context, sub push.Subscription, payload []byte, ttl time.Duration) error
}

// MessageTTL is how long a push service may hold a reminder for a phone
// that is offline; after that it is stale and better dropped.
const MessageTTL = time.Hour

// Reminders fires the daily reminder slots.
type Reminders struct {
	data   ReminderData
	sender Sender
	log    *slog.Logger
	now    func() time.Time
}

// NewReminders returns the reminder job. now is time.Now in production.
func NewReminders(data ReminderData, sender Sender, log *slog.Logger, now func() time.Time) *Reminders {
	return &Reminders{data: data, sender: sender, log: log, now: now}
}

// Tick fires every enabled slot that is due and not yet sent today. The
// log row is written first, so a crash or a second server cannot send twice.
func (r *Reminders) Tick(ctx context.Context) error {
	loc, slots, err := r.data.ReminderPlan(ctx)
	if err != nil {
		return err
	}
	local := r.now().In(loc)
	date := recur.Date(local)
	for _, s := range slots {
		if !s.Enabled || !slotDue(local, s.AtLocal) {
			continue
		}
		claimed, err := r.data.ClaimReminder(ctx, s.Slot, date)
		if err != nil {
			return err
		}
		if !claimed {
			continue
		}
		goals, planned, err := r.data.DayState(ctx, date)
		if err != nil {
			return err
		}
		msg := BuildMessage(s.Kind, s.Slot, date, goals, planned)
		sent, failed := Deliver(ctx, r.data, r.sender, r.log, msg)
		r.log.Info("reminder", "slot", s.Slot, "kind", s.Kind, "sent", sent, "failed", failed)
	}
	return nil
}

// Deliver sends msg to every subscription and records each outcome.
func Deliver(ctx context.Context, data ReminderData, sender Sender, log *slog.Logger, msg Message) (sent, failed int) {
	subs, err := data.PushSubs(ctx)
	if err != nil {
		log.Error("list subscriptions", "err", err)
		return 0, 0
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Error("encode message", "err", err)
		return 0, len(subs)
	}
	for _, p := range subs {
		err := sender.Send(ctx, push.Subscription{Endpoint: p.Endpoint, P256DH: p.P256DH, Auth: p.Auth}, payload, MessageTTL)
		gone := errors.Is(err, push.ErrGone)
		switch {
		case err == nil:
			sent++
		case gone:
			failed++
			log.Info("push subscription gone; removed", "label", p.Label)
		default:
			failed++
			// The endpoint is a secret; the label says which device it was.
			log.Warn("push failed", "label", p.Label, "err", err)
		}
		if err == nil || gone {
			if err := data.PushDelivered(ctx, p.Endpoint, gone); err != nil {
				log.Error("record push result", "err", err)
			}
		}
	}
	return sent, failed
}

// TestMessage is what "Send a test now" shows.
func TestMessage() Message {
	return Message{Title: "Homebase", Body: "Reminders reach this device.", URL: "/", Tag: "test"}
}
