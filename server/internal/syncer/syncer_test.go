package syncer

import (
	"testing"
	"time"

	"homebase/internal/store"
)

func TestNewIDIsValid(t *testing.T) {
	seen := map[string]bool{}
	for range 1000 {
		got := NewID(start)
		if !ValidID(got) {
			t.Fatalf("NewID() = %q is not a valid ULID", got)
		}
		if seen[got] {
			t.Fatalf("NewID() repeated %q", got)
		}
		seen[got] = true
	}
	a, b := NewID(start), NewID(start.Add(time.Millisecond))
	if a[:10] >= b[:10] {
		t.Errorf("time prefix does not sort: %q then %q", a, b)
	}
}

func TestValidID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"01HZZZZZZZZZZZZZZZZZZZ0001", true},
		{"01hzzzzzzzzzzzzzzzzzzz0001", false},
		{"01HZZZZZZZZZZZZZZZZZZZ000I", false}, // I is not Crockford
		{"81HZZZZZZZZZZZZZZZZZZZ0001", false}, // overflows 128 bits
		{"01HZZ", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := ValidID(tt.in); got != tt.want {
			t.Errorf("ValidID(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestInitSeedsSettingsAndRemindersOnce(t *testing.T) {
	e := newEnv(t)
	res, err := e.s.Pull(e.ctx, 0, MaxPullLimit)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Reminders) != 3 {
		t.Fatalf("first pull reminders = %d, want 3", len(res.Reminders))
	}
	if err := e.s.Init(e.ctx); err != nil {
		t.Fatal(err)
	}
	again, err := e.s.Pull(e.ctx, 0, MaxPullLimit)
	if err != nil {
		t.Fatal(err)
	}
	if again.Rev != res.Rev {
		t.Errorf("second Init changed rev from %d to %d", res.Rev, again.Rev)
	}
	me, err := e.s.Me(e.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if me.TZ != "UTC" || me.WeekStart != 1 || me.Rev != res.Rev {
		t.Errorf("Me() = %+v", me)
	}
}

func TestConflictResolution(t *testing.T) {
	tests := []struct {
		name       string
		firstAt    time.Duration
		secondAt   time.Duration
		wantTitle  string
		wantReason string
	}{
		{"later edit wins", 0, time.Second, "second", ""},
		{"earlier edit loses", time.Second, 0, "first", ReasonStale},
		{"same timestamp is a repeat, not a change", time.Second, time.Second, "first", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			e.mustApply(upsert("tasks", id(1), e.at(tt.firstAt), taskRow("first")))
			res := e.push(upsert("tasks", id(1), e.at(tt.secondAt), taskRow("second")))

			if got := rejectedReason(res, id(1)); got != tt.wantReason {
				t.Errorf("reason = %q, want %q", got, tt.wantReason)
			}
			if got := e.task(id(1)).Title; got != tt.wantTitle {
				t.Errorf("title = %q, want %q", got, tt.wantTitle)
			}
		})
	}
}

func TestRejectedRowIsReturnedForUndo(t *testing.T) {
	e := newEnv(t)
	first := e.mustApply(upsert("tasks", id(1), e.at(time.Second), taskRow("server copy")))
	res, err := e.s.Push(e.ctx, first.Rev, []Op{upsert("tasks", id(1), e.at(0), taskRow("stale copy"))})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Changes.Tasks) != 1 || res.Changes.Tasks[0].Title != "server copy" {
		t.Fatalf("changes.tasks = %+v, want the server copy", res.Changes.Tasks)
	}
}

func TestDeleteVersusEdit(t *testing.T) {
	tests := []struct {
		name        string
		deleteAt    time.Duration
		editAt      time.Duration
		deleteFirst bool
		wantDeleted bool
		wantTitle   string
	}{
		{"delete then later edit revives", time.Second, 2 * time.Second, true, false, "edited"},
		{"delete then earlier edit stays deleted", 2 * time.Second, time.Second, true, true, "original"},
		{"edit then later delete deletes", 2 * time.Second, time.Second, false, true, "edited"},
		{"edit then earlier delete is stale", time.Second, 2 * time.Second, false, false, "edited"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			e.mustApply(upsert("tasks", id(1), e.at(0), taskRow("original")))
			deleteOp := del("tasks", id(1), e.at(tt.deleteAt), "")
			editOp := upsert("tasks", id(1), e.at(tt.editAt), taskRow("edited"))
			if tt.deleteFirst {
				e.push(deleteOp)
				e.push(editOp)
			} else {
				e.push(editOp)
				e.push(deleteOp)
			}

			got := e.task(id(1))
			if (got.DeletedAt != nil) != tt.wantDeleted {
				t.Errorf("deleted = %v, want %v", got.DeletedAt != nil, tt.wantDeleted)
			}
			if got.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", got.Title, tt.wantTitle)
			}
		})
	}
}

func TestDeleteUnknownOrDeletedRowIsApplied(t *testing.T) {
	e := newEnv(t)
	res := e.push(del("tasks", id(9), e.at(0), ""))
	if len(res.Applied) != 1 {
		t.Fatalf("delete of unknown row: %+v", res)
	}
	e.mustApply(upsert("tasks", id(1), e.at(0), taskRow("a")), del("tasks", id(1), e.at(time.Second), ""))
	res = e.push(del("tasks", id(1), e.at(time.Second), ""))
	if len(res.Applied) != 1 {
		t.Fatalf("second delete: %+v", res)
	}
}

func TestCascade(t *testing.T) {
	const (
		goal = 1
		proj = 2
		pt   = 3 // task in the project
		gt   = 4 // task directly under the goal
		pa   = 5 // attachment on the project
		ta   = 6 // attachment on the project's task
		ga   = 7 // attachment on the goal
	)
	note := func(owner func(*store.Attachment)) store.Attachment {
		a := store.Attachment{Kind: "note", Body: ptr("text")}
		owner(&a)
		return a
	}

	tests := []struct {
		name         string
		op           func(e *env) Op
		wantDeleted  []int
		wantLive     []int
		wantNoParent []int // tasks or projects that lost their parent
	}{
		{
			name:        "delete project with contents",
			op:          func(e *env) Op { return del("projects", id(proj), e.at(time.Minute), CascadeDelete) },
			wantDeleted: []int{proj, pt, pa, ta},
			wantLive:    []int{goal, gt, ga},
		},
		{
			name:         "delete project, keep tasks",
			op:           func(e *env) Op { return del("projects", id(proj), e.at(time.Minute), CascadeDetach) },
			wantDeleted:  []int{proj, pa},
			wantLive:     []int{goal, pt, ta, gt, ga},
			wantNoParent: []int{pt},
		},
		{
			name:         "delete project without cascade detaches",
			op:           func(e *env) Op { return del("projects", id(proj), e.at(time.Minute), "") },
			wantDeleted:  []int{proj, pa},
			wantLive:     []int{pt, ta},
			wantNoParent: []int{pt},
		},
		{
			name:        "delete goal with contents",
			op:          func(e *env) Op { return del("goals", id(goal), e.at(time.Minute), CascadeDelete) },
			wantDeleted: []int{goal, proj, pt, gt, pa, ta, ga},
		},
		{
			name:         "delete goal, keep contents",
			op:           func(e *env) Op { return del("goals", id(goal), e.at(time.Minute), CascadeDetach) },
			wantDeleted:  []int{goal, ga},
			wantLive:     []int{proj, pt, gt, pa, ta},
			wantNoParent: []int{proj, gt},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			p := projRow("Project")
			p.GoalID = ptr(id(goal))
			inProject := taskRow("In project")
			inProject.ProjectID = ptr(id(proj))
			underGoal := taskRow("Under goal")
			underGoal.GoalID = ptr(id(goal))
			e.mustApply(
				upsert("goals", id(goal), e.at(0), goalRow("Goal")),
				upsert("projects", id(proj), e.at(0), p),
				upsert("tasks", id(pt), e.at(0), inProject),
				upsert("tasks", id(gt), e.at(0), underGoal),
				upsert("attachments", id(pa), e.at(0), note(func(a *store.Attachment) { a.ProjectID = ptr(id(proj)) })),
				upsert("attachments", id(ta), e.at(0), note(func(a *store.Attachment) { a.TaskID = ptr(id(pt)) })),
				upsert("attachments", id(ga), e.at(0), note(func(a *store.Attachment) { a.GoalID = ptr(id(goal)) })),
			)

			e.mustApply(tt.op(e))

			res, err := e.s.Pull(e.ctx, 0, MaxPullLimit)
			if err != nil {
				t.Fatal(err)
			}
			deleted := map[string]bool{}
			parent := map[string]bool{}
			for _, r := range res.Goals {
				deleted[r.ID] = r.DeletedAt != nil
			}
			for _, r := range res.Projects {
				deleted[r.ID] = r.DeletedAt != nil
				parent[r.ID] = r.GoalID != nil
			}
			for _, r := range res.Tasks {
				deleted[r.ID] = r.DeletedAt != nil
				parent[r.ID] = r.GoalID != nil || r.ProjectID != nil
			}
			for _, r := range res.Attachments {
				deleted[r.ID] = r.DeletedAt != nil
			}

			for _, n := range tt.wantDeleted {
				if !deleted[id(n)] {
					t.Errorf("row %d is live, want deleted", n)
				}
			}
			for _, n := range tt.wantLive {
				if deleted[id(n)] {
					t.Errorf("row %d is deleted, want live", n)
				}
			}
			for _, n := range tt.wantNoParent {
				if parent[id(n)] {
					t.Errorf("row %d still has a parent", n)
				}
			}
		})
	}
}

func TestThreePerDay(t *testing.T) {
	today := "2026-03-04"
	planned := func(status string, day string) store.Task {
		return store.Task{Title: "t", Status: status, PlannedOn: ptr(day), PlanRank: ptr(1.0)}
	}

	tests := []struct {
		name       string
		existing   []store.Task
		next       store.Task
		wantReason string
	}{
		{"third fits", []store.Task{planned("open", today), planned("open", today)}, planned("open", today), ""},
		{"fourth is rejected", []store.Task{planned("open", today), planned("open", today), planned("open", today)}, planned("open", today), ReasonDayFull},
		{"done tasks still count", []store.Task{planned("done", today), planned("done", today), planned("open", today)}, planned("open", today), ReasonDayFull},
		{"dropped tasks do not count", []store.Task{planned("dropped", today), planned("open", today), planned("open", today)}, planned("open", today), ""},
		{"future days have the same limit", []store.Task{planned("open", "2026-03-10"), planned("open", "2026-03-10"), planned("open", "2026-03-10")}, planned("open", "2026-03-10"), ReasonDayFull},
		{"another day is free", []store.Task{planned("open", today), planned("open", today), planned("open", today)}, planned("open", "2026-03-05"), ""},
		{"a dropped task may stay on a full day", []store.Task{planned("open", today), planned("open", today), planned("open", today)}, planned("dropped", today), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			var ops []Op
			for i, row := range tt.existing {
				ops = append(ops, upsert("tasks", id(i+1), e.at(0), row))
			}
			e.mustApply(ops...)

			res := e.push(upsert("tasks", id(99), e.at(time.Second), tt.next))
			if got := rejectedReason(res, id(99)); got != tt.wantReason {
				t.Errorf("reason = %q, want %q", got, tt.wantReason)
			}
		})
	}
}

func TestEditingAPlannedTaskOnAFullDayIsAllowed(t *testing.T) {
	e := newEnv(t)
	for i := 1; i <= 3; i++ {
		e.mustApply(upsert("tasks", id(i), e.at(0), store.Task{Title: "t", Status: "open", PlannedOn: ptr("2026-03-04")}))
	}
	done := store.Task{Title: "renamed and done", Status: "done", PlannedOn: ptr("2026-03-04")}
	e.mustApply(upsert("tasks", id(2), e.at(time.Second), done))
}

func TestServerOwnedFields(t *testing.T) {
	e := newEnv(t)
	row := store.Task{Title: "t", Status: "open", Slipped: 7, Rev: 999, DeletedAt: ptr(int64(5))}
	e.mustApply(upsert("tasks", id(1), e.at(0), row))
	got := e.task(id(1))
	if got.Slipped != 0 || got.Rev == 999 || got.DeletedAt != nil {
		t.Errorf("server-owned fields taken from client: %+v", got)
	}

	e.mustApply(upsert("tasks", id(1), e.at(time.Second), store.Task{Title: "t", Status: "done"}))
	got = e.task(id(1))
	if got.DoneAt == nil || *got.DoneAt != e.at(time.Second) {
		t.Errorf("doneAt = %v, want filled with the op time", deref(got.DoneAt))
	}

	e.mustApply(upsert("tasks", id(1), e.at(2*time.Second), store.Task{Title: "t", Status: "open", DoneAt: ptr(int64(1))}))
	if got := e.task(id(1)); got.DoneAt != nil {
		t.Errorf("doneAt = %v on an open task, want null", deref(got.DoneAt))
	}
}

func TestClockClamp(t *testing.T) {
	e := newEnv(t)
	e.mustApply(upsert("tasks", id(1), e.at(time.Hour), taskRow("from the future")))
	if got := e.task(id(1)).UpdatedAt; got != e.at(0) {
		t.Errorf("updatedAt = %d, want clamped to %d", got, e.at(0))
	}
	// Within the allowed skew the client's time is kept.
	e.mustApply(upsert("tasks", id(2), e.at(4*time.Minute), taskRow("slightly ahead")))
	if got := e.task(id(2)).UpdatedAt; got != e.at(4*time.Minute) {
		t.Errorf("updatedAt = %d, want %d", got, e.at(4*time.Minute))
	}
}

func TestIdempotentRepush(t *testing.T) {
	e := newEnv(t)
	e.mustApply(
		upsert("repeats", id(3), e.at(0), store.Repeat{Freq: "day", Every: 1, Mode: "after_done"}),
		upsert("tasks", id(4), e.at(0), store.Task{Title: "chore", Status: "open", RepeatID: ptr(id(3))}),
		upsert("tasks", id(5), e.at(0), taskRow("to delete")),
	)

	// One op per row, as the client's outbox coalesces them. A lost response
	// makes the client send the same batch again.
	batch := []Op{
		upsert("goals", id(1), e.at(time.Second), goalRow("Goal")),
		upsert("tasks", id(2), e.at(time.Second), store.Task{Title: "t", Status: "open", GoalID: ptr(id(1))}),
		upsert("tasks", id(4), e.at(time.Second), store.Task{Title: "chore", Status: "done", RepeatID: ptr(id(3))}),
		del("tasks", id(5), e.at(time.Second), ""),
	}
	first := e.mustApply(batch...)
	second := e.mustApply(batch...)

	if second.Rev != first.Rev {
		t.Errorf("re-push moved rev from %d to %d", first.Rev, second.Rev)
	}
	if len(second.Applied) != len(batch) {
		t.Errorf("re-push applied %d ops, want %d", len(second.Applied), len(batch))
	}
	if n := len(e.tasksWithRepeat(id(3))); n != 2 {
		t.Errorf("repeat instances = %d, want 2 (done one and its successor)", n)
	}
}

func TestValidation(t *testing.T) {
	longTitle := ""
	for range 301 {
		longTitle += "a"
	}

	tests := []struct {
		name string
		op   Op
	}{
		{"bad id", upsert("tasks", "not-a-ulid", ms(start), taskRow("t"))},
		{"no updatedAt", upsert("tasks", id(1), 0, taskRow("t"))},
		{"unknown table", upsert("sessions", id(1), ms(start), taskRow("t"))},
		{"unknown op", Op{Op: "merge", Table: "tasks", ID: id(1), UpdatedAt: ms(start)}},
		{"no row", Op{Op: "upsert", Table: "tasks", ID: id(1), UpdatedAt: ms(start)}},
		{"row id mismatch", upsert("tasks", id(1), ms(start), store.Task{ID: id(2), Title: "t", Status: "open"})},
		{"empty title", upsert("tasks", id(1), ms(start), taskRow(""))},
		{"blank title", upsert("tasks", id(1), ms(start), taskRow("   "))},
		{"title too long", upsert("tasks", id(1), ms(start), taskRow(longTitle))},
		{"bad status", upsert("tasks", id(1), ms(start), store.Task{Title: "t", Status: "paused"})},
		{"bad date", upsert("tasks", id(1), ms(start), store.Task{Title: "t", Status: "open", Due: ptr("2026-02-30")})},
		{"project and goal", upsert("tasks", id(1), ms(start), store.Task{Title: "t", Status: "open", GoalID: ptr(id(2)), ProjectID: ptr(id(3))})},
		{"missing project", upsert("tasks", id(1), ms(start), store.Task{Title: "t", Status: "open", ProjectID: ptr(id(3))})},
		{"bad repeat", upsert("repeats", id(1), ms(start), store.Repeat{Freq: "year", Every: 1, Mode: "fixed"})},
		{"bad weekdays", upsert("repeats", id(1), ms(start), store.Repeat{Freq: "week", Every: 1, Mode: "fixed", Weekdays: ptr(128)})},
		{"link without url", upsert("attachments", id(1), ms(start), store.Attachment{Kind: "link"})},
		{"javascript link", upsert("attachments", id(1), ms(start), store.Attachment{Kind: "link", URL: ptr("javascript:alert(1)")})},
		{"attachment with two owners", upsert("attachments", id(1), ms(start), store.Attachment{Kind: "note", Body: ptr("x"), TaskID: ptr(id(2)), GoalID: ptr(id(3))})},
		{"attachment without owner", upsert("attachments", id(1), ms(start), store.Attachment{Kind: "note", Body: ptr("x")})},
		{"file not uploaded", upsert("attachments", id(1), ms(start), store.Attachment{Kind: "file", FileSHA: ptr("aa"), TaskID: ptr(id(2))})},
		{"cascade on upsert", Op{Op: "upsert", Table: "tasks", ID: id(1), Row: []byte(`{"title":"t"}`), UpdatedAt: ms(start), Cascade: "delete"}},
		{"cascade on task delete", del("tasks", id(1), ms(start), CascadeDelete)},
		{"bad cascade", del("projects", id(1), ms(start), "explode")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			res := e.push(tt.op)
			if len(res.Rejected) != 1 || res.Rejected[0].Reason != "invalid" {
				t.Errorf("result = %+v, want one invalid rejection", res)
			}
		})
	}
}

func TestRejectedOpDoesNotUndoOthers(t *testing.T) {
	e := newEnv(t)
	res := e.push(
		upsert("tasks", id(1), e.at(0), taskRow("kept")),
		upsert("tasks", id(2), e.at(0), store.Task{Title: "t", Status: "open", ProjectID: ptr(id(9))}),
		upsert("tasks", id(3), e.at(0), taskRow("kept too")),
	)
	if len(res.Applied) != 2 || len(res.Rejected) != 1 {
		t.Fatalf("result = %+v", res)
	}
	if len(res.Changes.Tasks) != 2 {
		t.Errorf("changes.tasks = %d, want 2", len(res.Changes.Tasks))
	}
}

func TestPullPages(t *testing.T) {
	e := newEnv(t)
	var ops []Op
	for i := 1; i <= 7; i++ {
		ops = append(ops, upsert("tasks", id(i), e.at(0), taskRow("t")))
	}
	ops = append(ops, upsert("goals", id(8), e.at(0), goalRow("g")))
	e.mustApply(ops...)

	seen := map[string]bool{}
	since := int64(0)
	for pages := 0; ; pages++ {
		if pages > 10 {
			t.Fatal("pull did not finish")
		}
		res, err := e.s.Pull(e.ctx, since, 3)
		if err != nil {
			t.Fatal(err)
		}
		n := len(res.Goals) + len(res.Tasks) + len(res.Reminders)
		if n > 3 {
			t.Fatalf("page has %d rows, limit 3", n)
		}
		for _, r := range res.Tasks {
			seen[r.ID] = true
		}
		for _, r := range res.Goals {
			seen[r.ID] = true
		}
		since = res.Rev
		if !res.More {
			break
		}
	}
	if len(seen) != 8 {
		t.Errorf("saw %d rows across pages, want 8", len(seen))
	}
}

func TestPullArguments(t *testing.T) {
	e := newEnv(t)
	for _, tt := range []struct {
		since int64
		limit int
	}{{-1, 10}, {0, 0}, {0, MaxPullLimit + 1}} {
		if _, err := e.s.Pull(e.ctx, tt.since, tt.limit); err == nil {
			t.Errorf("Pull(%d, %d) = nil error", tt.since, tt.limit)
		}
	}
}

func TestClaimReminderOncePerDay(t *testing.T) {
	e := newEnv(t)
	tests := []struct {
		slot int
		date string
		want bool
	}{
		{1, "2026-03-04", true},
		{1, "2026-03-04", false},
		{2, "2026-03-04", true},
		{1, "2026-03-05", true},
	}
	for _, tt := range tests {
		got, err := e.s.ClaimReminder(e.ctx, tt.slot, tt.date)
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("ClaimReminder(%d, %s) = %v, want %v", tt.slot, tt.date, got, tt.want)
		}
	}
}
