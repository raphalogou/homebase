package syncer

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"homebase/internal/store"
)

// device simulates a client: a local copy of every row, an outbox coalesced
// per row, and the last revision it has seen.
type device struct {
	e      *env
	rev    int64
	rows   map[string]any // table/id -> row
	outbox []Op
}

func newDevice(e *env) *device { return &device{e: e, rows: map[string]any{}} }

func (d *device) queue(op Op) {
	for i, o := range d.outbox {
		if o.Table == op.Table && o.ID == op.ID {
			d.outbox = append(d.outbox[:i], d.outbox[i+1:]...)
			break
		}
	}
	d.outbox = append(d.outbox, op)
}

func (d *device) sync() {
	d.e.t.Helper()
	res, err := d.e.s.Push(d.e.ctx, d.rev, d.outbox)
	if err != nil {
		d.e.t.Fatalf("push: %v", err)
	}
	d.outbox = nil
	d.take(res.Changes)
	d.rev = res.Rev
	for {
		page, err := d.e.s.Pull(d.e.ctx, d.rev, 2)
		if err != nil {
			d.e.t.Fatalf("pull: %v", err)
		}
		d.take(page.Changes)
		d.rev = page.Rev
		if !page.More {
			return
		}
	}
}

func (d *device) take(c store.Changes) {
	for _, r := range c.Goals {
		d.rows["goals/"+r.ID] = r
	}
	for _, r := range c.Projects {
		d.rows["projects/"+r.ID] = r
	}
	for _, r := range c.Tasks {
		d.rows["tasks/"+r.ID] = r
	}
	for _, r := range c.Repeats {
		d.rows["repeats/"+r.ID] = r
	}
	for _, r := range c.Attachments {
		d.rows["attachments/"+r.ID] = r
	}
}

func serverRows(e *env) map[string]any {
	d := newDevice(e)
	res, err := e.s.Pull(e.ctx, 0, MaxPullLimit)
	if err != nil {
		e.t.Fatal(err)
	}
	d.take(res.Changes)
	return d.rows
}

func TestTwoDevicesConverge(t *testing.T) {
	e := newEnv(t)
	phone, laptop := newDevice(e), newDevice(e)
	tick := func() int64 { e.now = e.now.Add(time.Second); return ms(e.now) }

	// Shared starting point, made on the laptop.
	laptop.queue(upsert("goals", id(1), tick(), goalRow("Learn Go")))
	laptop.queue(upsert("projects", id(2), tick(), store.Project{Title: "Budgeting app", Status: "open", GoalID: ptr(id(1))}))
	laptop.queue(upsert("tasks", id(3), tick(), store.Task{Title: "Write the importer", Status: "open", ProjectID: ptr(id(2))}))
	laptop.queue(upsert("tasks", id(4), tick(), taskRow("Call the bank")))
	laptop.queue(upsert("repeats", id(5), tick(), store.Repeat{Freq: "week", Every: 1, Mode: "after_done"}))
	laptop.queue(upsert("tasks", id(6), tick(), store.Task{Title: "Take out the bins", Status: "open", RepeatID: ptr(id(5))}))
	laptop.sync()
	phone.sync()

	// Both go offline and edit.
	phoneEdit := tick()
	laptopEdit := tick()
	phone.queue(upsert("tasks", id(3), phoneEdit, store.Task{Title: "Write the CSV importer", Status: "open", ProjectID: ptr(id(2))}))
	laptop.queue(upsert("tasks", id(3), laptopEdit, store.Task{Title: "Write the OFX importer", Status: "open", ProjectID: ptr(id(2))}))

	phone.queue(del("tasks", id(4), tick(), ""))
	laptop.queue(upsert("tasks", id(4), tick(), store.Task{Title: "Call the bank about fees", Status: "open"}))

	phone.queue(upsert("tasks", id(6), tick(), store.Task{Title: "Take out the bins", Status: "done", RepeatID: ptr(id(5))}))

	for i := 10; i < 13; i++ {
		phone.queue(upsert("tasks", id(i), tick(), store.Task{Title: "Phone plan", Status: "open", PlannedOn: ptr("2026-03-05")}))
	}
	laptop.queue(upsert("tasks", id(20), tick(), store.Task{Title: "Laptop plan", Status: "open", PlannedOn: ptr("2026-03-05")}))

	laptop.queue(upsert("attachments", id(30), tick(), store.Attachment{Kind: "note", Body: ptr("Use encoding/csv"), ProjectID: ptr(id(2))}))
	laptop.queue(del("projects", id(2), tick(), CascadeDetach))

	// They come back online in turn; the first one syncs again at the end.
	phone.sync()
	laptop.sync()
	phone.sync()

	want := serverRows(e)
	if !reflect.DeepEqual(phone.rows, want) {
		t.Errorf("phone differs from server:\n%s", diff(phone.rows, want))
	}
	if !reflect.DeepEqual(laptop.rows, want) {
		t.Errorf("laptop differs from server:\n%s", diff(laptop.rows, want))
	}

	// And the outcome follows the rules.
	if got := want["tasks/"+id(3)].(store.Task); got.Title != "Write the OFX importer" || got.ProjectID != nil {
		t.Errorf("task 3 = %+v, want the later edit, detached", got)
	}
	if got := want["tasks/"+id(4)].(store.Task); got.DeletedAt != nil || got.Title != "Call the bank about fees" {
		t.Errorf("task 4 = %+v, want revived by the later edit", got)
	}
	planned := 0
	for _, r := range want {
		if task, ok := r.(store.Task); ok && deref(task.PlannedOn) == "2026-03-05" && task.DeletedAt == nil {
			planned++
		}
	}
	if planned != 3 {
		t.Errorf("planned for 2026-03-05 = %d, want 3", planned)
	}
	if n := len(e.tasksWithRepeat(id(5))); n != 2 {
		t.Errorf("bins instances = %d, want 2", n)
	}
}

func diff(a, b map[string]any) string {
	out := ""
	for k, v := range b {
		if !reflect.DeepEqual(a[k], v) {
			ja, _ := json.Marshal(a[k])
			jb, _ := json.Marshal(v)
			out += k + "\n  device: " + string(ja) + "\n  server: " + string(jb) + "\n"
		}
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			out += k + " only on device\n"
		}
	}
	return out
}
