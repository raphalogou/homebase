package syncer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"homebase/internal/db"
	"homebase/internal/migrate"
	"homebase/internal/store"
	"homebase/migrations"
)

// env is one server with an in-memory database and a clock the test moves.
type env struct {
	t   *testing.T
	ctx context.Context
	db  *sql.DB
	st  store.Store
	s   *Syncer
	now time.Time
}

// noon on Wednesday 2026-03-04, UTC.
var start = time.Date(2026, 3, 4, 12, 0, 0, 0, time.UTC)

func newEnv(t *testing.T) *env {
	t.Helper()
	d, err := db.OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	ctx := context.Background()
	if err := migrate.Run(ctx, d, migrations.FS, time.Now); err != nil {
		t.Fatal(err)
	}
	e := &env{t: t, ctx: ctx, db: d, st: store.NewSQLite(d), now: start}
	e.s = New(e.st, func() time.Time { return e.now })
	if err := e.s.Init(ctx); err != nil {
		t.Fatal(err)
	}
	return e
}

func id(n int) string { return fmt.Sprintf("01HZZZZZZZZZZZZZZZZZZZ%04d", n) }

func ptr[T any](v T) *T { return &v }

func ms(t time.Time) int64 { return t.UnixMilli() }

// at returns the clock plus d, in milliseconds.
func (e *env) at(d time.Duration) int64 { return ms(e.now.Add(d)) }

func upsert(table, rowID string, at int64, row any) Op {
	b, err := json.Marshal(row)
	if err != nil {
		panic(err)
	}
	return Op{Op: "upsert", Table: table, ID: rowID, Row: b, UpdatedAt: at}
}

func del(table, rowID string, at int64, cascade string) Op {
	return Op{Op: "delete", Table: table, ID: rowID, UpdatedAt: at, Cascade: cascade}
}

func taskRow(title string) store.Task    { return store.Task{Title: title, Status: "open"} }
func goalRow(title string) store.Goal    { return store.Goal{Title: title, Status: "open"} }
func projRow(title string) store.Project { return store.Project{Title: title, Status: "open"} }

func (e *env) push(ops ...Op) PushResult {
	e.t.Helper()
	res, err := e.s.Push(e.ctx, 0, ops)
	if err != nil {
		e.t.Fatalf("push: %v", err)
	}
	return res
}

// mustApply pushes ops and fails if any is rejected.
func (e *env) mustApply(ops ...Op) PushResult {
	e.t.Helper()
	res := e.push(ops...)
	if len(res.Rejected) > 0 {
		e.t.Fatalf("rejected: %+v", res.Rejected)
	}
	return res
}

func (e *env) task(taskID string) store.Task {
	e.t.Helper()
	var v store.Task
	err := e.st.Tx(e.ctx, func(tx store.Tx) error {
		var err error
		v, err = tx.Task(taskID)
		return err
	})
	if err != nil {
		e.t.Fatalf("task %s: %v", taskID, err)
	}
	return v
}

func (e *env) project(projectID string) store.Project {
	e.t.Helper()
	var v store.Project
	err := e.st.Tx(e.ctx, func(tx store.Tx) error {
		var err error
		v, err = tx.Project(projectID)
		return err
	})
	if err != nil {
		e.t.Fatalf("project %s: %v", projectID, err)
	}
	return v
}

func (e *env) attachment(attID string) store.Attachment {
	e.t.Helper()
	var v store.Attachment
	err := e.st.Tx(e.ctx, func(tx store.Tx) error {
		var err error
		v, err = tx.Attachment(attID)
		return err
	})
	if err != nil {
		e.t.Fatalf("attachment %s: %v", attID, err)
	}
	return v
}

// tasksWithRepeat lists live tasks carrying repeatID.
func (e *env) tasksWithRepeat(repeatID string) []store.Task {
	e.t.Helper()
	res, err := e.s.Pull(e.ctx, 0, MaxPullLimit)
	if err != nil {
		e.t.Fatal(err)
	}
	var out []store.Task
	for _, t := range res.Tasks {
		if t.RepeatID != nil && *t.RepeatID == repeatID && t.DeletedAt == nil {
			out = append(out, t)
		}
	}
	return out
}

func (e *env) setTZ(tz string) {
	e.t.Helper()
	if _, err := e.db.Exec(`UPDATE settings SET tz = ?`, tz); err != nil {
		e.t.Fatal(err)
	}
}

func rejectedReason(res PushResult, rowID string) string {
	for _, r := range res.Rejected {
		if r.ID == rowID {
			return r.Reason
		}
	}
	return ""
}

func deref[T any](p *T) any {
	if p == nil {
		return nil
	}
	return *p
}
