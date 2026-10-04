package review

import (
	"testing"
	"time"

	"homebase/internal/store"
)

// Wednesday 4 March 2026, noon UTC.
var now = time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)

func ms(t time.Time) int64      { return t.UnixMilli() }
func ago(d time.Duration) int64 { return ms(now.Add(-d)) }
func p[T any](v T) *T           { return &v }
func days(n int) time.Duration  { return time.Duration(n) * 24 * time.Hour }

func TestWeekStartOf(t *testing.T) {
	tests := []struct {
		name      string
		now       time.Time
		tz        string
		weekStart int
		want      string
	}{
		{"Monday weeks", now, "UTC", 1, "2026-03-02"},
		{"Sunday weeks", now, "UTC", 0, "2026-03-01"},
		{"on the first day", time.Date(2026, 3, 2, 8, 0, 0, 0, time.UTC), "UTC", 1, "2026-03-02"},
		{"Sunday in a Monday week", time.Date(2026, 3, 8, 8, 0, 0, 0, time.UTC), "UTC", 1, "2026-03-02"},
		// 23:30 UTC on Sunday 1 March is already Monday 2 March in Tokyo.
		{"the user's zone decides", time.Date(2026, 3, 1, 23, 30, 0, 0, time.UTC), "Asia/Tokyo", 1, "2026-03-02"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc, _ := time.LoadLocation(tt.tz)
			if got := WeekStartOf(tt.now, loc, tt.weekStart); got != tt.want {
				t.Errorf("WeekStartOf = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestDoneThisWeek(t *testing.T) {
	goals := []store.Goal{
		{ID: "G1", Title: "Run a half marathon", Status: "open"},
		{ID: "G2", Title: "Learn Go", Status: "open"},
		{ID: "G3", Title: "Paused goal", Status: "paused"},
	}
	projects := []store.Project{{ID: "P1", GoalID: p("G2"), Title: "Budgeting app", Status: "open"}}
	done := func(at time.Time, goal, project *string) store.Task {
		return store.Task{Status: "done", DoneAt: p(ms(at)), GoalID: goal, ProjectID: project}
	}
	tasks := []store.Task{
		done(now.Add(-time.Hour), p("G1"), nil),
		done(now.Add(-days(1)), p("G1"), nil),
		done(now.Add(-days(2)), nil, p("P1")), // Monday: through the project to G2
		done(now.Add(-days(3)), p("G1"), nil), // Sunday: last week
		done(now.Add(-time.Hour), nil, nil),   // standalone counts overall only
		{Status: "open", GoalID: p("G1")},
	}
	s := Compute(Input{Now: now, Loc: time.UTC, WeekStart: 1, Goals: goals, Projects: projects, Tasks: tasks})

	if s.WeekStart != "2026-03-02" || s.DoneCount != 4 {
		t.Fatalf("week %s, done %d; want 2026-03-02 and 4", s.WeekStart, s.DoneCount)
	}
	if len(s.DoneByGoal) != 2 || s.DoneByGoal[0].GoalID != "G1" || s.DoneByGoal[0].Count != 2 || s.DoneByGoal[1].Count != 1 {
		t.Errorf("doneByGoal = %+v", s.DoneByGoal)
	}
	if len(s.GoalsWithNothing) != 0 {
		t.Errorf("goalsWithNothing = %+v, want none (a paused goal is not asked about)", s.GoalsWithNothing)
	}
}

func TestQuiet(t *testing.T) {
	old := ago(days(30))
	tests := []struct {
		name      string
		goals     []store.Goal
		projects  []store.Project
		tasks     []store.Task
		wantQuiet []string
	}{
		{
			name:      "project with open tasks and nothing for 30 days",
			projects:  []store.Project{{ID: "P", Title: "p", Status: "open", UpdatedAt: old}},
			tasks:     []store.Task{{ProjectID: p("P"), Status: "open", CreatedAt: old, UpdatedAt: old}},
			wantQuiet: []string{"project P"},
		},
		{
			name:     "a task edited last week keeps the project awake",
			projects: []store.Project{{ID: "P", Title: "p", Status: "open", UpdatedAt: old}},
			tasks:    []store.Task{{ProjectID: p("P"), Status: "open", CreatedAt: old, UpdatedAt: ago(days(7))}},
		},
		{
			name:     "editing the project itself keeps it awake",
			projects: []store.Project{{ID: "P", Title: "p", Status: "open", UpdatedAt: ago(days(2))}},
			tasks:    []store.Task{{ProjectID: p("P"), Status: "open", CreatedAt: old, UpdatedAt: old}},
		},
		{
			name:     "a project with no open tasks is not asked about",
			projects: []store.Project{{ID: "P", Title: "p", Status: "open", UpdatedAt: old}},
			tasks:    []store.Task{{ProjectID: p("P"), Status: "done", DoneAt: p(old), CreatedAt: old, UpdatedAt: old}},
		},
		{
			name:     "a paused project is not asked about",
			projects: []store.Project{{ID: "P", Title: "p", Status: "paused", UpdatedAt: old}},
			tasks:    []store.Task{{ProjectID: p("P"), Status: "open", CreatedAt: old, UpdatedAt: old}},
		},
		{
			name:      "goal untouched for 30 days",
			goals:     []store.Goal{{ID: "G", Title: "g", Status: "open", UpdatedAt: old}},
			wantQuiet: []string{"goal G"},
		},
		{
			name:  "a task of the goal done this week keeps it awake",
			goals: []store.Goal{{ID: "G", Title: "g", Status: "open", UpdatedAt: old}},
			tasks: []store.Task{{GoalID: p("G"), Status: "done", DoneAt: p(ago(days(2))), CreatedAt: old, UpdatedAt: old}},
		},
		{
			name:     "a task planned through a project keeps the goal awake",
			goals:    []store.Goal{{ID: "G", Title: "g", Status: "open", UpdatedAt: old}},
			projects: []store.Project{{ID: "P", GoalID: p("G"), Title: "p", Status: "open", UpdatedAt: ago(days(1))}},
			tasks:    []store.Task{{ProjectID: p("P"), Status: "open", PlannedOn: p("2026-03-05"), CreatedAt: old, UpdatedAt: old}},
		},
		{
			name:      "merely editing a goal's task does not count for the goal",
			goals:     []store.Goal{{ID: "G", Title: "g", Status: "open", UpdatedAt: old}},
			tasks:     []store.Task{{GoalID: p("G"), Status: "open", CreatedAt: old, UpdatedAt: ago(days(1))}},
			wantQuiet: []string{"goal G"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Compute(Input{Now: now, Loc: time.UTC, WeekStart: 1, Goals: tt.goals, Projects: tt.projects, Tasks: tt.tasks})
			var got []string
			for _, q := range s.Quiet {
				got = append(got, q.Kind+" "+q.ID)
			}
			if len(got) != len(tt.wantQuiet) || (len(got) > 0 && got[0] != tt.wantQuiet[0]) {
				t.Errorf("quiet = %v, want %v", got, tt.wantQuiet)
			}
		})
	}
}
