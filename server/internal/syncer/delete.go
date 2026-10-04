package syncer

import (
	"homebase/internal/store"
)

// Cascade choices for deleting a goal or a project.
const (
	CascadeDelete = "delete"
	CascadeDetach = "detach"
)

// delete tombstones a row. Children of a goal or project are deleted or
// detached according to op.Cascade; without one, they are detached, which
// never loses a task. Attachments always go with their owner.
func (a *applier) delete(op Op, at int64) error {
	switch op.Cascade {
	case "", CascadeDetach, CascadeDelete:
	default:
		return invalid(`cascade must be "delete" or "detach".`)
	}
	if op.Cascade != "" && op.Table != "goals" && op.Table != "projects" {
		return invalid("cascade applies only to goals and projects.")
	}

	switch op.Table {
	case "goals":
		g, found, err := lookup(a.tx.Goal, op.ID)
		if skip, err := deleteCheck(found, err, g.UpdatedAt, g.DeletedAt, at); skip || err != nil {
			return err
		}
		return a.deleteGoal(g, at, op.Cascade == CascadeDelete)
	case "projects":
		p, found, err := lookup(a.tx.Project, op.ID)
		if skip, err := deleteCheck(found, err, p.UpdatedAt, p.DeletedAt, at); skip || err != nil {
			return err
		}
		return a.deleteProject(p, at, op.Cascade == CascadeDelete)
	case "tasks":
		t, found, err := lookup(a.tx.Task, op.ID)
		if skip, err := deleteCheck(found, err, t.UpdatedAt, t.DeletedAt, at); skip || err != nil {
			return err
		}
		return a.deleteTask(t, at)
	case "repeats":
		r, found, err := lookup(a.tx.Repeat, op.ID)
		if skip, err := deleteCheck(found, err, r.UpdatedAt, r.DeletedAt, at); skip || err != nil {
			return err
		}
		r.DeletedAt, r.UpdatedAt = &at, at
		if r.Rev, err = a.tx.NextRev(); err != nil {
			return err
		}
		return a.tx.PutRepeat(r)
	case "attachments":
		t, found, err := lookup(a.tx.Attachment, op.ID)
		if skip, err := deleteCheck(found, err, t.UpdatedAt, t.DeletedAt, at); skip || err != nil {
			return err
		}
		return a.deleteAttachment(t, at)
	default:
		return invalid("table must be goals, projects, tasks, repeats or attachments.")
	}
}

// deleteCheck applies last write wins to a delete. Deleting a row the server
// never saw, or one already deleted, is a no-op that still counts as applied.
func deleteCheck(found bool, err error, updatedAt int64, deletedAt *int64, at int64) (skip bool, _ error) {
	if err != nil {
		return false, err
	}
	if !found || deletedAt != nil {
		return true, nil
	}
	if updatedAt > at {
		return false, stale()
	}
	return false, nil
}

func (a *applier) deleteGoal(g store.Goal, at int64, withContents bool) error {
	projects, err := a.tx.ProjectsOfGoal(g.ID)
	if err != nil {
		return err
	}
	for _, p := range projects {
		if withContents {
			err = a.deleteProject(p, at, true)
		} else {
			p.GoalID = nil
			err = a.touchProject(p, at)
		}
		if err != nil {
			return err
		}
	}

	tasks, err := a.tx.TasksOfGoal(g.ID)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if withContents {
			err = a.deleteTask(t, at)
		} else {
			t.GoalID = nil
			err = a.touchTask(t, at)
		}
		if err != nil {
			return err
		}
	}

	if err := a.deleteAttachmentsOf(store.OwnerGoal, g.ID, at); err != nil {
		return err
	}
	g.DeletedAt, g.UpdatedAt = &at, max(at, g.UpdatedAt)
	if g.Rev, err = a.tx.NextRev(); err != nil {
		return err
	}
	return a.tx.PutGoal(g)
}

func (a *applier) deleteProject(p store.Project, at int64, withContents bool) error {
	tasks, err := a.tx.TasksOfProject(p.ID)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if withContents {
			err = a.deleteTask(t, at)
		} else {
			t.ProjectID = nil
			err = a.touchTask(t, at)
		}
		if err != nil {
			return err
		}
	}

	if err := a.deleteAttachmentsOf(store.OwnerProject, p.ID, at); err != nil {
		return err
	}
	p.DeletedAt, p.UpdatedAt = &at, max(at, p.UpdatedAt)
	if p.Rev, err = a.tx.NextRev(); err != nil {
		return err
	}
	return a.tx.PutProject(p)
}

func (a *applier) deleteTask(t store.Task, at int64) error {
	if err := a.deleteAttachmentsOf(store.OwnerTask, t.ID, at); err != nil {
		return err
	}
	t.DeletedAt = &at
	return a.touchTask(t, at)
}

func (a *applier) deleteAttachmentsOf(kind store.OwnerKind, ownerID string, at int64) error {
	list, err := a.tx.AttachmentsOf(kind, ownerID)
	if err != nil {
		return err
	}
	for _, t := range list {
		if err := a.deleteAttachment(t, at); err != nil {
			return err
		}
	}
	return nil
}

func (a *applier) deleteAttachment(t store.Attachment, at int64) error {
	var err error
	t.DeletedAt, t.UpdatedAt = &at, max(at, t.UpdatedAt)
	if t.Rev, err = a.tx.NextRev(); err != nil {
		return err
	}
	return a.tx.PutAttachment(t)
}

// touchProject and touchTask write a row changed by a cascade. updated_at
// never moves backwards, so a cascade cannot undo a newer edit's timestamp.
func (a *applier) touchProject(p store.Project, at int64) error {
	var err error
	p.UpdatedAt = max(at, p.UpdatedAt)
	if p.Rev, err = a.tx.NextRev(); err != nil {
		return err
	}
	return a.tx.PutProject(p)
}

func (a *applier) touchTask(t store.Task, at int64) error {
	var err error
	t.UpdatedAt = max(at, t.UpdatedAt)
	if t.Rev, err = a.tx.NextRev(); err != nil {
		return err
	}
	return a.tx.PutTask(t)
}
