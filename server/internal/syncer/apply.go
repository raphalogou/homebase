package syncer

import (
	"bytes"
	"encoding/json"
	"errors"
	"time"

	"homebase/internal/apperr"
	"homebase/internal/recur"
	"homebase/internal/store"
)

// Rejection reasons beyond the error codes.
const (
	ReasonStale   = "stale"
	ReasonDayFull = "day_full"
)

// MaxPerDay is the number of tasks one day can hold, done ones included.
const MaxPerDay = 3

type applier struct {
	tx    store.Tx
	now   time.Time
	ms    int64
	loc   *time.Location
	today string
}

func newApplier(tx store.Tx, now time.Time) (*applier, error) {
	st, err := tx.Settings()
	if err != nil {
		return nil, err
	}
	loc := location(st.TZ)
	return &applier{tx: tx, now: now, ms: now.UnixMilli(), loc: loc, today: recur.Date(now.In(loc))}, nil
}

func invalid(msg string) error { return apperr.New(apperr.Invalid, msg) }

func stale() error {
	return apperr.WithReason(apperr.Invalid, ReasonStale, "A newer version of this row exists.")
}

func (a *applier) apply(op Op) error {
	if !ValidID(op.ID) {
		return invalid("id must be a ULID.")
	}
	if op.UpdatedAt <= 0 {
		return invalid("updatedAt is required.")
	}
	at := op.UpdatedAt
	// A clock far ahead would otherwise win every future conflict.
	if at > a.ms+clockSkew.Milliseconds() {
		at = a.ms
	}

	switch op.Op {
	case "upsert":
		if op.Cascade != "" {
			return invalid("cascade applies only to delete.")
		}
		return a.upsert(op, at)
	case "delete":
		return a.delete(op, at)
	default:
		return invalid(`op must be "upsert" or "delete".`)
	}
}

// decodeRow reads the row of an upsert. Unknown fields are ignored so that a
// client may send back a whole row, server-owned fields included.
func decodeRow(op Op, v any) error {
	if len(op.Row) == 0 || bytes.Equal(op.Row, []byte("null")) {
		return invalid("upsert needs a row.")
	}
	if err := json.Unmarshal(op.Row, v); err != nil {
		return invalid("row is not valid for this table.")
	}
	return nil
}

func checkRowID(op Op, rowID string) error {
	if rowID != "" && rowID != op.ID {
		return invalid("row id does not match op id.")
	}
	return nil
}

// lww decides an upsert or delete against the stored row. found is false for
// a new row. It returns skip=true when the same change was already applied,
// which makes re-pushing an op harmless.
func lww(found bool, storedUpdatedAt, at int64) (skip bool, err error) {
	if !found {
		return false, nil
	}
	if storedUpdatedAt > at {
		return false, stale()
	}
	return storedUpdatedAt == at, nil
}

func lookup[T any](get func(string) (T, error), id string) (T, bool, error) {
	v, err := get(id)
	if errors.Is(err, store.ErrNotFound) {
		return v, false, nil
	}
	return v, err == nil, err
}

func createdAt(found bool, stored, sent, at int64) int64 {
	if found {
		return stored
	}
	if sent > 0 && sent <= at {
		return sent
	}
	return at
}

func (a *applier) upsert(op Op, at int64) error {
	switch op.Table {
	case "goals":
		return a.upsertGoal(op, at)
	case "projects":
		return a.upsertProject(op, at)
	case "tasks":
		return a.upsertTask(op, at)
	case "repeats":
		return a.upsertRepeat(op, at)
	case "attachments":
		return a.upsertAttachment(op, at)
	default:
		return invalid("table must be goals, projects, tasks, repeats or attachments.")
	}
}

func (a *applier) upsertGoal(op Op, at int64) error {
	var g store.Goal
	if err := decodeRow(op, &g); err != nil {
		return err
	}
	if err := checkRowID(op, g.ID); err != nil {
		return err
	}
	if err := validateGoal(g); err != nil {
		return err
	}
	old, found, err := lookup(a.tx.Goal, op.ID)
	if err != nil {
		return err
	}
	if skip, err := lww(found, old.UpdatedAt, at); skip || err != nil {
		return err
	}

	g.ID = op.ID
	g.CreatedAt = createdAt(found, old.CreatedAt, g.CreatedAt, at)
	g.UpdatedAt = at
	g.DeletedAt = nil
	if g.Rev, err = a.tx.NextRev(); err != nil {
		return err
	}
	return a.tx.PutGoal(g)
}

func (a *applier) upsertProject(op Op, at int64) error {
	var p store.Project
	if err := decodeRow(op, &p); err != nil {
		return err
	}
	if err := checkRowID(op, p.ID); err != nil {
		return err
	}
	if err := validateProject(p); err != nil {
		return err
	}
	if err := a.mustExist("goal", a.goalExists, p.GoalID); err != nil {
		return err
	}
	old, found, err := lookup(a.tx.Project, op.ID)
	if err != nil {
		return err
	}
	if skip, err := lww(found, old.UpdatedAt, at); skip || err != nil {
		return err
	}

	p.ID = op.ID
	p.CreatedAt = createdAt(found, old.CreatedAt, p.CreatedAt, at)
	p.UpdatedAt = at
	p.DeletedAt = nil
	if p.Rev, err = a.tx.NextRev(); err != nil {
		return err
	}
	return a.tx.PutProject(p)
}

func (a *applier) upsertRepeat(op Op, at int64) error {
	var r store.Repeat
	if err := decodeRow(op, &r); err != nil {
		return err
	}
	if err := checkRowID(op, r.ID); err != nil {
		return err
	}
	if r.Every == 0 {
		r.Every = 1
	}
	if r.Mode == "" {
		r.Mode = "after_done"
	}
	if err := validateRepeat(r); err != nil {
		return err
	}
	old, found, err := lookup(a.tx.Repeat, op.ID)
	if err != nil {
		return err
	}
	if skip, err := lww(found, old.UpdatedAt, at); skip || err != nil {
		return err
	}

	r.ID = op.ID
	r.CreatedAt = createdAt(found, old.CreatedAt, r.CreatedAt, at)
	r.UpdatedAt = at
	r.DeletedAt = nil
	if r.Rev, err = a.tx.NextRev(); err != nil {
		return err
	}
	return a.tx.PutRepeat(r)
}

func (a *applier) upsertAttachment(op Op, at int64) error {
	var t store.Attachment
	if err := decodeRow(op, &t); err != nil {
		return err
	}
	if err := checkRowID(op, t.ID); err != nil {
		return err
	}
	if err := validateAttachment(t); err != nil {
		return err
	}
	if err := a.mustExist("goal", a.goalExists, t.GoalID); err != nil {
		return err
	}
	if err := a.mustExist("project", a.projectExists, t.ProjectID); err != nil {
		return err
	}
	if err := a.mustExist("task", a.taskExists, t.TaskID); err != nil {
		return err
	}
	if err := a.mustExist("file", a.tx.FileExists, t.FileSHA); err != nil {
		return err
	}
	old, found, err := lookup(a.tx.Attachment, op.ID)
	if err != nil {
		return err
	}
	if skip, err := lww(found, old.UpdatedAt, at); skip || err != nil {
		return err
	}

	t.ID = op.ID
	t.CreatedAt = createdAt(found, old.CreatedAt, t.CreatedAt, at)
	t.UpdatedAt = at
	t.DeletedAt = nil
	if t.Rev, err = a.tx.NextRev(); err != nil {
		return err
	}
	return a.tx.PutAttachment(t)
}

func (a *applier) upsertTask(op Op, at int64) error {
	var t store.Task
	if err := decodeRow(op, &t); err != nil {
		return err
	}
	if err := checkRowID(op, t.ID); err != nil {
		return err
	}
	if t.Status == "" {
		t.Status = "open"
	}
	if err := validateTask(t); err != nil {
		return err
	}
	if err := a.mustExist("goal", a.goalExists, t.GoalID); err != nil {
		return err
	}
	if err := a.mustExist("project", a.projectExists, t.ProjectID); err != nil {
		return err
	}
	if err := a.mustExist("repeat", a.repeatExists, t.RepeatID); err != nil {
		return err
	}
	old, found, err := lookup(a.tx.Task, op.ID)
	if err != nil {
		return err
	}
	if skip, err := lww(found, old.UpdatedAt, at); skip || err != nil {
		return err
	}

	t.ID = op.ID
	t.CreatedAt = createdAt(found, old.CreatedAt, t.CreatedAt, at)
	t.UpdatedAt = at
	t.DeletedAt = nil
	t.Slipped = old.Slipped // server-owned; zero for a new row

	switch {
	case t.Status != "done":
		t.DoneAt = nil
	case found && old.Status == "done" && old.DoneAt != nil:
		t.DoneAt = old.DoneAt
	case t.DoneAt == nil || *t.DoneAt <= 0 || *t.DoneAt > at:
		t.DoneAt = &at
	}

	// An open task is never planned in the past. This catches an edit made
	// before midnight that arrives after the rollover; the slip was counted
	// by the rollover already, or the plan never reached the server.
	if t.Status == "open" && t.PlannedOn != nil && *t.PlannedOn < a.today {
		t.PlannedOn = nil
		t.PlanRank = nil
	}
	if t.PlannedOn == nil {
		t.PlanRank = nil
	}

	if t.PlannedOn != nil && t.Status != "dropped" {
		n, err := a.tx.CountPlanned(*t.PlannedOn, t.ID)
		if err != nil {
			return err
		}
		if n >= MaxPerDay {
			return apperr.WithReason(apperr.Invalid, ReasonDayFull, "That day already has three tasks.")
		}
	}

	if t.Rev, err = a.tx.NextRev(); err != nil {
		return err
	}
	if err := a.tx.PutTask(t); err != nil {
		return err
	}

	finished := t.Status == "done" || t.Status == "dropped"
	if found && old.DeletedAt == nil && old.Status == "open" && finished && t.RepeatID != nil {
		return a.spawnNext(t, at)
	}
	return nil
}

// spawnNext creates the next instance of a repeating task. Completing or
// dropping a task both count; dropping is skipping this time.
func (a *applier) spawnNext(done store.Task, at int64) error {
	r, err := a.tx.Repeat(*done.RepeatID)
	if err != nil {
		return err
	}
	if r.DeletedAt != nil {
		return nil
	}
	if open, err := a.tx.HasOpenInstance(r.ID, done.ID); err != nil || open {
		return err
	}

	rule := recur.Rule{Freq: r.Freq, Every: r.Every, Mode: r.Mode}
	if r.Weekdays != nil {
		rule.Weekdays = *r.Weekdays
	}
	if r.Until != nil {
		rule.Until = *r.Until
	}
	oldDue := ""
	if done.Due != nil {
		oldDue = *done.Due
	}
	finishedAt := at
	if done.DoneAt != nil {
		finishedAt = *done.DoneAt
	}
	doneOn := recur.Date(time.UnixMilli(finishedAt).In(a.loc))

	due, ok, err := recur.Next(rule, oldDue, doneOn, a.today)
	if err != nil || !ok {
		return err
	}

	next := store.Task{
		ID:        NewID(a.now),
		ProjectID: done.ProjectID,
		GoalID:    done.GoalID,
		Title:     done.Title,
		Notes:     done.Notes,
		Status:    "open",
		Due:       &due,
		RepeatID:  done.RepeatID,
		CreatedAt: at,
		UpdatedAt: at,
	}
	if next.Rev, err = a.tx.NextRev(); err != nil {
		return err
	}
	return a.tx.PutTask(next)
}

func (a *applier) mustExist(what string, exists func(string) (bool, error), id *string) error {
	if id == nil {
		return nil
	}
	ok, err := exists(*id)
	if err != nil {
		return err
	}
	if !ok {
		return invalid("The " + what + " this row points to does not exist.")
	}
	return nil
}

func exists[T any](get func(string) (T, error)) func(string) (bool, error) {
	return func(id string) (bool, error) {
		_, found, err := lookup(get, id)
		return found, err
	}
}

func (a *applier) goalExists(id string) (bool, error)    { return exists(a.tx.Goal)(id) }
func (a *applier) projectExists(id string) (bool, error) { return exists(a.tx.Project)(id) }
func (a *applier) taskExists(id string) (bool, error)    { return exists(a.tx.Task)(id) }
func (a *applier) repeatExists(id string) (bool, error)  { return exists(a.tx.Repeat)(id) }
