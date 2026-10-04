// Package sched runs the background jobs: the daily rollover now, reminders
// in Phase 4.
package sched

import (
	"context"
	"log/slog"
	"time"

	"homebase/internal/recur"
)

// Rollover is the part of the syncer the scheduler needs.
type Rollover interface {
	LocalNow(ctx context.Context) (time.Time, error)
	RolloverDone(ctx context.Context, today string) (bool, error)
	Rollover(ctx context.Context) error
}

// RolloverAt is the local time of day the rollover runs.
const RolloverAt = 5 * time.Minute

// Run checks once a tick until ctx ends. tick is a minute in production.
func Run(ctx context.Context, log *slog.Logger, r Rollover, tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()
	for {
		if err := runRollover(ctx, r); err != nil && ctx.Err() == nil {
			log.Error("rollover", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func runRollover(ctx context.Context, r Rollover) error {
	local, err := r.LocalNow(ctx)
	if err != nil {
		return err
	}
	done, err := r.RolloverDone(ctx, recur.Date(local))
	if err != nil {
		return err
	}
	if !rolloverDue(local, done) {
		return nil
	}
	return r.Rollover(ctx)
}

// rolloverDue reports whether the rollover should run now. Before 00:05 it
// waits; a sync in that window still runs it, which is fine.
func rolloverDue(local time.Time, doneToday bool) bool {
	if doneToday {
		return false
	}
	y, m, d := local.Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, local.Location())
	return local.Sub(midnight) >= RolloverAt
}
