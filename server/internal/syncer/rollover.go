package syncer

import (
	"context"
	"time"

	"homebase/internal/recur"
	"homebase/internal/store"
)

// Rollover runs the daily rollover if it has not run today. The scheduler
// calls it shortly after midnight; every sync calls it too, in case the
// server was off.
func (s *Syncer) Rollover(ctx context.Context) error {
	return s.store.Tx(ctx, func(tx store.Tx) error {
		return rollover(tx, s.now())
	})
}

// LocalNow returns the current time in the user's zone.
func (s *Syncer) LocalNow(ctx context.Context) (time.Time, error) {
	var loc *time.Location
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		st, err := tx.Settings()
		if err != nil {
			return err
		}
		loc = location(st.TZ)
		return nil
	})
	if err != nil {
		return time.Time{}, err
	}
	return s.now().In(loc), nil
}

// RolloverDone reports whether the rollover already ran for the local day.
func (s *Syncer) RolloverDone(ctx context.Context, today string) (bool, error) {
	var done bool
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		st, err := tx.Settings()
		if err != nil {
			return err
		}
		done = st.LastRollover != nil && *st.LastRollover >= today
		return nil
	})
	return done, err
}

// rollover returns every open task planned before today to the pile:
// planned_on and plan_rank cleared, slipped increased by one.
//
// It leaves updated_at alone. Stamping it would make the rollover win over an
// edit made offline before midnight (say, ticking the task done at 23:59).
// The new rev is enough for every device to pull the change.
func rollover(tx store.Tx, now time.Time) error {
	st, err := tx.Settings()
	if err != nil {
		return err
	}
	today := recur.Date(now.In(location(st.TZ)))
	if st.LastRollover != nil && *st.LastRollover >= today {
		return nil
	}

	tasks, err := tx.OpenTasksPlannedBefore(today)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		t.PlannedOn = nil
		t.PlanRank = nil
		t.Slipped++
		if t.Rev, err = tx.NextRev(); err != nil {
			return err
		}
		if err := tx.PutTask(t); err != nil {
			return err
		}
	}
	return tx.SetLastRollover(today)
}
