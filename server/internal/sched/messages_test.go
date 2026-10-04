package sched

import (
	"testing"
	"time"

	"homebase/internal/store"
)

func TestSlotDue(t *testing.T) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		t.Fatal(err)
	}
	at := func(h, m int) time.Time { return time.Date(2026, 3, 4, h, m, 0, 0, paris) }
	tests := []struct {
		name string
		now  time.Time
		slot string
		want bool
	}{
		{"a minute early", at(7, 59), "08:00", false},
		{"on time", at(8, 0), "08:00", true},
		{"late but within grace", at(9, 29), "08:00", true},
		{"past the grace", at(9, 30), "08:00", false},
		{"evening slot in the morning", at(8, 0), "20:00", false},
		{"bad time", at(8, 0), "8am", false},
		// The clocks go forward at 02:00 on 29 March in Paris; 08:00 is still 08:00.
		{"after a clock change", time.Date(2026, 3, 29, 8, 0, 0, 0, paris), "08:00", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slotDue(tt.now, tt.slot); got != tt.want {
				t.Errorf("slotDue(%s, %s) = %v, want %v", tt.now.Format("15:04"), tt.slot, got, tt.want)
			}
		})
	}
}

func TestBuildMessage(t *testing.T) {
	goals := []store.Goal{{Title: "Run a half marathon in spring"}, {Title: "Learn Go well enough to build tools"}}
	open := func(title string) store.Task { return store.Task{Title: title, Status: "open"} }
	done := func(title string) store.Task { return store.Task{Title: title, Status: "done"} }

	tests := []struct {
		name      string
		kind      string
		date      string
		goals     []store.Goal
		planned   []store.Task
		wantTitle string
		wantBody  string
	}{
		{"focus lists the open picks", "focus", "2026-03-04", goals,
			[]store.Task{open("Easy 5 km run"), open("Fix the import bug")},
			"Run a half marathon in spring", "Today: Easy 5 km run; Fix the import bug."},
		{"focus goal turns over the next day", "focus", "2026-03-05", goals, nil,
			"Learn Go well enough to build tools", "Choose your three for today."},
		{"focus without goals", "focus", "2026-03-04", nil, []store.Task{done("Book the physio")},
			"Today", "All done for today."},
		{"checkin with progress", "checkin", "2026-03-04", goals,
			[]store.Task{done("Easy 5 km run"), open("Fix the import bug")},
			"Midday check", "1 of 2 done. Next: Fix the import bug."},
		{"checkin all done", "checkin", "2026-03-04", goals, []store.Task{done("Easy 5 km run")},
			"Midday check", "All done for today."},
		{"checkin nothing planned", "checkin", "2026-03-04", goals, nil,
			"Midday check", "Nothing planned yet. Choose your three for today."},
		{"wrap with some left", "wrap", "2026-03-04", goals,
			[]store.Task{done("Easy 5 km run"), open("Fix the import bug")},
			"Close the day", "1 of 2 done. Move the rest or let it go."},
		{"wrap nothing planned", "wrap", "2026-03-04", goals, nil,
			"Close the day", "Nothing planned today. Capture what is on your mind."},
		{"wrap all done", "wrap", "2026-03-04", goals, []store.Task{done("a"), done("b")},
			"Close the day", "All done for today."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := BuildMessage(tt.kind, 1, tt.date, tt.goals, tt.planned)
			if m.Title != tt.wantTitle || m.Body != tt.wantBody {
				t.Errorf("got %q / %q, want %q / %q", m.Title, m.Body, tt.wantTitle, tt.wantBody)
			}
			if m.URL != "/" || m.Tag != "reminder-1" || !m.Sync {
				t.Errorf("message = %+v, want it to open Today with the sync hint", m)
			}
		})
	}
}

func TestFocusGoalRotatesThroughAll(t *testing.T) {
	goals := []store.Goal{{Title: "A"}, {Title: "B"}, {Title: "C"}}
	seen := map[string]bool{}
	for _, d := range []string{"2026-03-04", "2026-03-05", "2026-03-06"} {
		seen[BuildMessage("focus", 1, d, goals, nil).Title] = true
	}
	if len(seen) != 3 {
		t.Errorf("three days showed %d goals, want 3", len(seen))
	}
}
