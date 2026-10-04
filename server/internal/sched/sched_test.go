package sched

import (
	"context"
	"testing"
	"time"
)

func TestRolloverDue(t *testing.T) {
	day := func(h, m int) time.Time { return time.Date(2026, 3, 5, h, m, 0, 0, time.UTC) }
	tests := []struct {
		name  string
		local time.Time
		done  bool
		want  bool
	}{
		{"just after midnight waits", day(0, 1), false, false},
		{"00:04:59 waits", day(0, 4).Add(59 * time.Second), false, false},
		{"00:05 runs", day(0, 5), false, true},
		{"afternoon runs a missed one", day(15, 0), false, true},
		{"already done today", day(0, 6), true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rolloverDue(tt.local, tt.done); got != tt.want {
				t.Errorf("rolloverDue() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeRollover struct {
	local time.Time
	done  bool
	runs  int
}

func (f *fakeRollover) LocalNow(context.Context) (time.Time, error)        { return f.local, nil }
func (f *fakeRollover) RolloverDone(context.Context, string) (bool, error) { return f.done, nil }
func (f *fakeRollover) Rollover(context.Context) error                     { f.runs++; f.done = true; return nil }

func TestRunRollover(t *testing.T) {
	tests := []struct {
		name     string
		local    time.Time
		done     bool
		wantRuns int
	}{
		{"runs when due", time.Date(2026, 3, 5, 0, 5, 0, 0, time.UTC), false, 1},
		{"skips before 00:05", time.Date(2026, 3, 5, 0, 2, 0, 0, time.UTC), false, 0},
		{"skips when done", time.Date(2026, 3, 5, 9, 0, 0, 0, time.UTC), true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeRollover{local: tt.local, done: tt.done}
			if err := runRollover(context.Background(), f); err != nil {
				t.Fatal(err)
			}
			if f.runs != tt.wantRuns {
				t.Errorf("runs = %d, want %d", f.runs, tt.wantRuns)
			}
		})
	}
}
