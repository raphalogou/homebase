// Package syncer applies the sync protocol and the business rules of
// docs/SPEC.md sections 3 and 5: last write wins, cascades, the limit of three
// tasks a day, recurrence and the daily rollover.
package syncer

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"homebase/internal/apperr"
	"homebase/internal/store"
)

const (
	// MaxOps bounds one push so a single request cannot hold the database
	// for long.
	MaxOps = 1000
	// MaxPullLimit caps the page size of a pull.
	MaxPullLimit = 1000
	// clockSkew is how far ahead of the server a client's updatedAt may be.
	clockSkew = 5 * time.Minute
)

// Syncer is the entry point for everything that writes synced rows.
type Syncer struct {
	store store.Store
	now   func() time.Time
}

// New returns a Syncer. now is time.Now in production.
func New(s store.Store, now func() time.Time) *Syncer {
	return &Syncer{store: s, now: now}
}

// Op is one change pushed by a client.
type Op struct {
	Op        string          `json:"op"`
	Table     string          `json:"table"`
	ID        string          `json:"id"`
	Row       json.RawMessage `json:"row,omitempty"`
	UpdatedAt int64           `json:"updatedAt"`
	Cascade   string          `json:"cascade,omitempty"`
}

// Rejection says why an op was not applied.
type Rejection struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

// PushResult is the answer to POST /api/sync.
type PushResult struct {
	Rev      int64         `json:"rev"`
	Applied  []string      `json:"applied"`
	Rejected []Rejection   `json:"rejected"`
	Changes  store.Changes `json:"changes"`
}

// PullResult is the answer to GET /api/sync.
type PullResult struct {
	Rev  int64 `json:"rev"`
	More bool  `json:"more"`
	store.Changes
}

// Me is the answer to GET /api/me.
type Me struct {
	TZ        string `json:"tz"`
	WeekStart int    `json:"weekStart"`
	Rev       int64  `json:"rev"`
}

// Init runs first-run setup: the settings row with a random calendar token,
// and a revision for each seeded reminder row so that a first pull (since=0)
// returns them. It does nothing on later runs.
func (s *Syncer) Init(ctx context.Context) error {
	return s.store.Tx(ctx, func(tx store.Tx) error {
		_, err := tx.Settings()
		if err == nil {
			return nil
		}
		if !errors.Is(err, store.ErrNotFound) {
			return err
		}

		if err := tx.InsertSettings(store.Settings{
			TZ:            "UTC",
			WeekStart:     1,
			CalendarToken: NewSecret(),
		}); err != nil {
			return fmt.Errorf("insert settings: %w", err)
		}

		reminders, err := tx.Reminders()
		if err != nil {
			return err
		}
		now := s.now().UnixMilli()
		for _, r := range reminders {
			if r.Rev, err = tx.NextRev(); err != nil {
				return err
			}
			r.UpdatedAt = now
			if err := tx.PutReminder(r); err != nil {
				return err
			}
		}
		return nil
	})
}

// NewSecret returns 32 random bytes, base64url without padding.
func NewSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // crypto/rand.Read never fails
	return base64.RawURLEncoding.EncodeToString(b)
}

// Me returns the settings a client needs at start.
func (s *Syncer) Me(ctx context.Context) (Me, error) {
	var me Me
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		st, err := tx.Settings()
		if err != nil {
			return err
		}
		rev, err := tx.Rev()
		if err != nil {
			return err
		}
		me = Me{TZ: st.TZ, WeekStart: st.WeekStart, Rev: rev}
		return nil
	})
	return me, err
}

// Pull returns rows changed after since, at most limit of them.
func (s *Syncer) Pull(ctx context.Context, since int64, limit int) (PullResult, error) {
	if since < 0 {
		return PullResult{}, apperr.New(apperr.Invalid, "since must not be negative.")
	}
	if limit <= 0 || limit > MaxPullLimit {
		return PullResult{}, apperr.Newf(apperr.Invalid, "limit must be between 1 and %d.", MaxPullLimit)
	}

	var res PullResult
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		now := s.now()
		if err := rollover(tx, now); err != nil {
			return err
		}
		c, more, last, err := tx.ChangesSince(since, limit)
		if err != nil {
			return err
		}
		rev := last
		if !more {
			if rev, err = tx.Rev(); err != nil {
				return err
			}
		}
		res = PullResult{Rev: rev, More: more, Changes: c}
		return nil
	})
	return res, err
}

// Push applies ops in order in one transaction. An op that breaks a rule is
// rejected without affecting the others; any other error aborts the push.
func (s *Syncer) Push(ctx context.Context, base int64, ops []Op) (PushResult, error) {
	if base < 0 {
		return PushResult{}, apperr.New(apperr.Invalid, "base must not be negative.")
	}
	if len(ops) > MaxOps {
		return PushResult{}, apperr.Newf(apperr.TooLarge, "At most %d ops per push.", MaxOps)
	}

	res := PushResult{Applied: []string{}, Rejected: []Rejection{}}
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		now := s.now()
		if err := rollover(tx, now); err != nil {
			return err
		}
		a, err := newApplier(tx, now)
		if err != nil {
			return err
		}

		var rejectedRows []tableID
		for _, op := range ops {
			err := tx.Savepoint(func() error { return a.apply(op) })
			if err == nil {
				res.Applied = append(res.Applied, op.ID)
				continue
			}
			e, ok := apperr.As(err)
			if !ok {
				return fmt.Errorf("op %s %s %s: %w", op.Op, op.Table, op.ID, err)
			}
			reason := e.Reason
			if reason == "" {
				reason = string(e.Code)
			}
			res.Rejected = append(res.Rejected, Rejection{ID: op.ID, Reason: reason})
			rejectedRows = append(rejectedRows, tableID{op.Table, op.ID})
		}

		if res.Changes, _, _, err = tx.ChangesSince(base, 0); err != nil {
			return err
		}
		if err := addCurrentRows(tx, &res.Changes, base, rejectedRows); err != nil {
			return err
		}
		res.Rev, err = tx.Rev()
		return err
	})
	if err != nil {
		return PushResult{}, err
	}
	return res, nil
}

type tableID struct{ table, id string }

// addCurrentRows adds the server's copy of each rejected row that the client
// would not otherwise receive, so it can undo its optimistic change.
func addCurrentRows(tx store.Tx, c *store.Changes, base int64, rows []tableID) error {
	seen := map[tableID]bool{}
	for _, r := range rows {
		if seen[r] {
			continue
		}
		seen[r] = true
		var err error
		switch r.table {
		case "goals":
			var v store.Goal
			if v, err = tx.Goal(r.id); err == nil && v.Rev <= base {
				c.Goals = append(c.Goals, v)
			}
		case "projects":
			var v store.Project
			if v, err = tx.Project(r.id); err == nil && v.Rev <= base {
				c.Projects = append(c.Projects, v)
			}
		case "tasks":
			var v store.Task
			if v, err = tx.Task(r.id); err == nil && v.Rev <= base {
				c.Tasks = append(c.Tasks, v)
			}
		case "repeats":
			var v store.Repeat
			if v, err = tx.Repeat(r.id); err == nil && v.Rev <= base {
				c.Repeats = append(c.Repeats, v)
			}
		case "attachments":
			var v store.Attachment
			if v, err = tx.Attachment(r.id); err == nil && v.Rev <= base {
				c.Attachments = append(c.Attachments, v)
			}
		}
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			return err
		}
	}
	return nil
}

// location returns the user's zone, falling back to UTC if the stored name
// cannot be loaded; settings are validated on write, so that is a bug guard.
func location(tz string) *time.Location {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}
