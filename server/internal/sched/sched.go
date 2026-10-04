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

// Orphans is what the file clean-up needs.
type Orphans interface {
	OrphanFiles(ctx context.Context) ([]string, error)
}

// Remover deletes a stored file's bytes, reporting whether any were there.
type Remover interface {
	Remove(sha string) (bool, error)
}

// CleanEvery is how often unused files are looked for. The spec asks for
// nightly; hourly is cheap and catches up after the server was off.
const CleanEvery = time.Hour

// Run checks once a tick until ctx ends. tick is a minute in production.
func Run(ctx context.Context, log *slog.Logger, r Rollover, o Orphans, rm Remover, tick time.Duration) {
	t := time.NewTicker(tick)
	defer t.Stop()
	var lastClean time.Time
	for {
		if err := runRollover(ctx, r); err != nil && ctx.Err() == nil {
			log.Error("rollover", "err", err)
		}
		if time.Since(lastClean) >= CleanEvery {
			lastClean = time.Now()
			n, err := cleanFiles(ctx, o, rm)
			if err != nil && ctx.Err() == nil {
				log.Error("file clean-up", "err", err)
			} else if n > 0 {
				log.Info("file clean-up", "removed", n)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func cleanFiles(ctx context.Context, o Orphans, rm Remover) (int, error) {
	list, err := o.OrphanFiles(ctx)
	if err != nil {
		return 0, err
	}
	// Rows of cleaned files stay (tombstones point at them), so the list
	// repeats; only count bytes actually removed.
	n := 0
	for _, sha := range list {
		removed, err := rm.Remove(sha)
		if err != nil {
			return n, err
		}
		if removed {
			n++
		}
	}
	return n, nil
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
