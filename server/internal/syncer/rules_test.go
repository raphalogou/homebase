package syncer

import (
	"testing"
	"time"

	"homebase/internal/apperr"
	"homebase/internal/store"
)

func TestRepeatSpawn(t *testing.T) {
	tests := []struct {
		name     string
		repeat   store.Repeat
		due      *string
		finishAs string
		clock    time.Time // when the task is finished
		tz       string
		wantDue  string // "" means no successor
	}{
		{
			name:     "after_done counts from completion",
			repeat:   store.Repeat{Freq: "day", Every: 3, Mode: "after_done"},
			due:      ptr("2026-03-01"),
			finishAs: "done",
			clock:    start,
			wantDue:  "2026-03-07",
		},
		{
			name:     "fixed counts from the old due",
			repeat:   store.Repeat{Freq: "week", Every: 1, Mode: "fixed"},
			due:      ptr("2026-03-02"),
			finishAs: "done",
			clock:    start,
			wantDue:  "2026-03-09",
		},
		{
			name:     "dropping skips to the next one",
			repeat:   store.Repeat{Freq: "week", Every: 1, Mode: "fixed", Weekdays: ptr(1 | 8)},
			due:      ptr("2026-03-02"),
			finishAs: "dropped",
			clock:    start,
			wantDue:  "2026-03-05",
		},
		{
			name:     "after_done monthly at month end",
			repeat:   store.Repeat{Freq: "month", Every: 1, Mode: "after_done"},
			finishAs: "done",
			clock:    time.Date(2026, 1, 31, 18, 0, 0, 0, time.UTC),
			wantDue:  "2026-02-28",
		},
		{
			name:     "fixed monthly from the 31st",
			repeat:   store.Repeat{Freq: "month", Every: 1, Mode: "fixed"},
			due:      ptr("2026-03-31"),
			finishAs: "done",
			clock:    time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC),
			wantDue:  "2026-04-30",
		},
		{
			name:     "completion day is local, not UTC",
			repeat:   store.Repeat{Freq: "day", Every: 1, Mode: "after_done"},
			finishAs: "done",
			// 23:30 UTC on 4 March is already 5 March in Tokyo.
			clock:   time.Date(2026, 3, 4, 23, 30, 0, 0, time.UTC),
			tz:      "Asia/Tokyo",
			wantDue: "2026-03-06",
		},
		{
			name:     "until ends the chain",
			repeat:   store.Repeat{Freq: "day", Every: 1, Mode: "fixed", Until: ptr("2026-03-04")},
			due:      ptr("2026-03-04"),
			finishAs: "done",
			clock:    start,
			wantDue:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			if tt.tz != "" {
				e.setTZ(tt.tz)
			}
			e.now = tt.clock
			attached := store.Attachment{Kind: "note", Body: ptr("not copied"), TaskID: ptr(id(2))}
			e.mustApply(
				upsert("goals", id(3), e.at(-time.Hour), goalRow("Keep the flat nice")),
				upsert("repeats", id(1), e.at(-time.Hour), tt.repeat),
				upsert("tasks", id(2), e.at(-time.Hour), store.Task{Title: "Water the plants", Notes: "Kitchen too",
					Status: "open", Due: tt.due, GoalID: ptr(id(3)), RepeatID: ptr(id(1))}),
				upsert("attachments", id(4), e.at(-time.Hour), attached),
			)

			e.mustApply(upsert("tasks", id(2), e.at(0), store.Task{Title: "Water the plants", Notes: "Kitchen too",
				Status: tt.finishAs, Due: tt.due, GoalID: ptr(id(3)), RepeatID: ptr(id(1)), PlannedOn: ptr(recurToday(e))}))

			var next []store.Task
			for _, task := range e.tasksWithRepeat(id(1)) {
				if task.ID != id(2) {
					next = append(next, task)
				}
			}
			if tt.wantDue == "" {
				if len(next) != 0 {
					t.Fatalf("successors = %+v, want none", next)
				}
				return
			}
			if len(next) != 1 {
				t.Fatalf("successors = %d, want 1", len(next))
			}
			n := next[0]
			if n.Due == nil || *n.Due != tt.wantDue {
				t.Errorf("due = %v, want %s", deref(n.Due), tt.wantDue)
			}
			if n.Title != "Water the plants" || n.Notes != "Kitchen too" || n.GoalID == nil || *n.GoalID != id(3) {
				t.Errorf("successor did not copy title, notes and goal: %+v", n)
			}
			if n.Status != "open" || n.PlannedOn != nil || n.Slipped != 0 || n.DoneAt != nil {
				t.Errorf("successor copied state it should not: %+v", n)
			}

			res, err := e.s.Pull(e.ctx, 0, MaxPullLimit)
			if err != nil {
				t.Fatal(err)
			}
			for _, a := range res.Attachments {
				if a.TaskID != nil && *a.TaskID == n.ID {
					t.Errorf("successor got an attachment: %+v", a)
				}
			}
		})
	}
}

func recurToday(e *env) string { return e.now.Format("2006-01-02") }

func TestRepeatSpawnGuards(t *testing.T) {
	tests := []struct {
		name  string
		setup func(e *env)
		want  int // live tasks with the repeat after finishing task 2
	}{
		{
			name:  "one successor only",
			setup: func(e *env) {},
			want:  2,
		},
		{
			name: "no successor while another instance is open",
			setup: func(e *env) {
				e.mustApply(upsert("tasks", id(5), e.at(-time.Hour), store.Task{Title: "Water", Status: "open", RepeatID: ptr(id(1))}))
			},
			want: 2, // task 2 (done) and task 5
		},
		{
			name: "no successor once the repeat is deleted",
			setup: func(e *env) {
				e.mustApply(del("repeats", id(1), e.at(-time.Minute), ""))
			},
			want: 1,
		},
		{
			name: "re-opening and finishing again does not stack instances",
			setup: func(e *env) {
				e.mustApply(upsert("tasks", id(2), e.at(-30*time.Minute), store.Task{Title: "Water", Status: "done", RepeatID: ptr(id(1))}))
				e.mustApply(upsert("tasks", id(2), e.at(-20*time.Minute), store.Task{Title: "Water", Status: "open", RepeatID: ptr(id(1))}))
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			e.mustApply(
				upsert("repeats", id(1), e.at(-time.Hour), store.Repeat{Freq: "day", Every: 1, Mode: "after_done"}),
				upsert("tasks", id(2), e.at(-time.Hour), store.Task{Title: "Water", Status: "open", RepeatID: ptr(id(1))}),
			)
			tt.setup(e)
			e.mustApply(upsert("tasks", id(2), e.at(0), store.Task{Title: "Water", Status: "done", RepeatID: ptr(id(1))}))
			if got := len(e.tasksWithRepeat(id(1))); got != tt.want {
				t.Errorf("live tasks with repeat = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRollover(t *testing.T) {
	type row struct {
		status    string
		plannedOn *string
		slipped   int
	}
	tests := []struct {
		name string
		task row
		want row
	}{
		{"yesterday's open task returns to the pile", row{"open", ptr("2026-03-04"), 0}, row{"open", nil, 1}},
		{"slips add up", row{"open", ptr("2026-03-01"), 0}, row{"open", nil, 1}},
		{"done task keeps its day", row{"done", ptr("2026-03-04"), 0}, row{"done", ptr("2026-03-04"), 0}},
		{"today's task stays", row{"open", ptr("2026-03-05"), 0}, row{"open", ptr("2026-03-05"), 0}},
		{"unplanned task is untouched", row{"open", nil, 0}, row{"open", nil, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			e.now = time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
			e.mustApply(upsert("tasks", id(1), e.at(0), store.Task{Title: "t", Status: tt.task.status,
				PlannedOn: tt.task.plannedOn, PlanRank: ptr(1.0)}))
			before := e.task(id(1))

			e.now = time.Date(2026, 3, 5, 0, 5, 0, 0, time.UTC)
			if err := e.s.Rollover(e.ctx); err != nil {
				t.Fatal(err)
			}
			got := e.task(id(1))
			if deref(got.PlannedOn) != deref(tt.want.plannedOn) || got.Slipped != tt.want.slipped {
				t.Errorf("after rollover plannedOn=%v slipped=%d, want %v %d",
					deref(got.PlannedOn), got.Slipped, deref(tt.want.plannedOn), tt.want.slipped)
			}
			if got.PlannedOn == nil && got.PlanRank != nil {
				t.Errorf("planRank = %v, want cleared", *got.PlanRank)
			}
			if got.UpdatedAt != before.UpdatedAt {
				t.Errorf("updatedAt changed from %d to %d", before.UpdatedAt, got.UpdatedAt)
			}
		})
	}
}

func TestRolloverRunsOncePerDayAndOnSync(t *testing.T) {
	e := newEnv(t)
	e.mustApply(upsert("tasks", id(1), e.at(0), store.Task{Title: "t", Status: "open", PlannedOn: ptr("2026-03-04")}))

	// The first sync of the next day runs the missed rollover.
	e.now = start.Add(24 * time.Hour)
	if _, err := e.s.Pull(e.ctx, 0, MaxPullLimit); err != nil {
		t.Fatal(err)
	}
	if got := e.task(id(1)); got.PlannedOn != nil || got.Slipped != 1 {
		t.Fatalf("after first sync: %+v", got)
	}

	// Planned again for today, a second rollover the same day changes nothing.
	e.mustApply(upsert("tasks", id(1), e.at(0), store.Task{Title: "t", Status: "open", PlannedOn: ptr("2026-03-05")}))
	if err := e.s.Rollover(e.ctx); err != nil {
		t.Fatal(err)
	}
	if got := e.task(id(1)); got.PlannedOn == nil || got.Slipped != 1 {
		t.Fatalf("second rollover on the same day changed the task: %+v", got)
	}

	// Two days later it slips again: slipped counts rollovers, not days.
	e.now = start.Add(72 * time.Hour)
	if err := e.s.Rollover(e.ctx); err != nil {
		t.Fatal(err)
	}
	if got := e.task(id(1)); got.Slipped != 2 {
		t.Fatalf("slipped = %d, want 2", got.Slipped)
	}
}

func TestRolloverUsesUserTimeZone(t *testing.T) {
	e := newEnv(t)
	e.setTZ("America/New_York")
	e.mustApply(upsert("tasks", id(1), e.at(0), store.Task{Title: "t", Status: "open", PlannedOn: ptr("2026-03-04")}))

	// 02:00 UTC on 5 March is still 4 March in New York.
	e.now = time.Date(2026, 3, 5, 2, 0, 0, 0, time.UTC)
	if err := e.s.Rollover(e.ctx); err != nil {
		t.Fatal(err)
	}
	if got := e.task(id(1)); got.PlannedOn == nil {
		t.Fatal("rolled over before local midnight")
	}
}

func TestOfflineEditFromBeforeMidnight(t *testing.T) {
	e := newEnv(t)
	e.mustApply(upsert("tasks", id(1), e.at(0), store.Task{Title: "t", Status: "open", PlannedOn: ptr("2026-03-04")}))
	lateEvening := ms(time.Date(2026, 3, 4, 23, 59, 0, 0, time.UTC))

	e.now = time.Date(2026, 3, 5, 0, 5, 0, 0, time.UTC)
	if err := e.s.Rollover(e.ctx); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		status      string
		wantPlanned *string
	}{
		// Ticking done at 23:59 must survive the rollover at 00:05.
		{"done at 23:59 is kept on its day", "done", ptr("2026-03-04")},
		// A plain edit lands, but the task does not go back to yesterday.
		{"open edit does not re-plan yesterday", "open", nil},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := e.push(upsert("tasks", id(1), lateEvening+int64(i), store.Task{Title: "edited", Status: tt.status,
				PlannedOn: ptr("2026-03-04")}))
			if len(res.Rejected) > 0 {
				t.Fatalf("rejected: %+v", res.Rejected)
			}
			got := e.task(id(1))
			if got.Title != "edited" || got.Status != tt.status || deref(got.PlannedOn) != deref(tt.wantPlanned) {
				t.Errorf("task = %+v", got)
			}
			if got.Slipped != 1 {
				t.Errorf("slipped = %d, want 1", got.Slipped)
			}
		})
	}
}

func TestPromote(t *testing.T) {
	e := newEnv(t)
	e.mustApply(
		upsert("goals", id(1), e.at(0), goalRow("Run a half marathon")),
		upsert("tasks", id(2), e.at(0), store.Task{Title: "Plan training", Notes: "Three runs a week", Status: "open", Due: ptr("2026-04-01")}),
		upsert("attachments", id(3), e.at(0), store.Attachment{Kind: "link", URL: ptr("https://example.com/plan"), TaskID: ptr(id(2))}),
		upsert("attachments", id(4), e.at(0), store.Attachment{Kind: "note", Body: ptr("Ask about shoes"), TaskID: ptr(id(2))}),
	)
	e.now = e.now.Add(time.Minute)

	res, err := e.s.Promote(e.ctx, id(2), ptr(id(1)))
	if err != nil {
		t.Fatal(err)
	}
	p := res.Project
	if p.Title != "Plan training" || p.Notes != "Three runs a week" || deref(p.GoalID) != id(1) || deref(p.Due) != "2026-04-01" {
		t.Errorf("project = %+v", p)
	}
	if res.RemovedTaskID != id(2) || e.task(id(2)).DeletedAt == nil {
		t.Errorf("task not tombstoned")
	}
	for _, n := range []int{3, 4} {
		a := e.attachment(id(n))
		if deref(a.ProjectID) != p.ID || a.TaskID != nil || a.DeletedAt != nil {
			t.Errorf("attachment %d = %+v, want moved to the project", n, a)
		}
	}
	if got := e.project(p.ID); got.Rev == 0 {
		t.Error("project has no rev")
	}
}

func TestPromoteErrors(t *testing.T) {
	tests := []struct {
		name   string
		task   string
		goal   *string
		wantNF bool
	}{
		{"unknown task", id(9), nil, true},
		{"unknown goal", id(2), ptr(id(9)), true},
		{"task already in a project", id(3), nil, false},
		{"bad id", "x", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			e.mustApply(
				upsert("projects", id(1), e.at(0), projRow("p")),
				upsert("tasks", id(2), e.at(0), taskRow("t")),
				upsert("tasks", id(3), e.at(0), store.Task{Title: "t", Status: "open", ProjectID: ptr(id(1))}),
			)
			_, err := e.s.Promote(e.ctx, tt.task, tt.goal)
			e2, ok := apperr.As(err)
			if !ok {
				t.Fatalf("err = %v, want an apperr", err)
			}
			if want := map[bool]apperr.Code{true: apperr.NotFound, false: apperr.Invalid}[tt.wantNF]; e2.Code != want {
				t.Errorf("code = %s, want %s", e2.Code, want)
			}
		})
	}
}
