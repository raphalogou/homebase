package syncer

import (
	"context"
	"errors"

	"homebase/internal/apperr"
	"homebase/internal/store"
)

// Promoted is the answer to POST /api/promote.
type Promoted struct {
	Project       store.Project `json:"project"`
	RemovedTaskID string        `json:"removedTaskId"`
}

// Promote turns a task into a project in one transaction: the project takes
// the task's title, notes and due date, the task's attachments move to it,
// and the task is tombstoned. goalID, if given, places the project under that
// goal; otherwise the task's own goal is kept.
func (s *Syncer) Promote(ctx context.Context, taskID string, goalID *string) (Promoted, error) {
	if !ValidID(taskID) || !validRef(goalID) {
		return Promoted{}, invalid("taskId and goalId must be ULIDs.")
	}

	var res Promoted
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		now := s.now()
		at := now.UnixMilli()

		task, err := tx.Task(taskID)
		if errors.Is(err, store.ErrNotFound) || (err == nil && task.DeletedAt != nil) {
			return apperr.New(apperr.NotFound, "No such task.")
		}
		if err != nil {
			return err
		}
		if task.ProjectID != nil {
			return invalid("This task is already in a project.")
		}

		goal := task.GoalID
		if goalID != nil {
			g, err := tx.Goal(*goalID)
			if errors.Is(err, store.ErrNotFound) || (err == nil && g.DeletedAt != nil) {
				return apperr.New(apperr.NotFound, "No such goal.")
			}
			if err != nil {
				return err
			}
			goal = goalID
		}

		p := store.Project{
			ID:        NewID(now),
			GoalID:    goal,
			Title:     task.Title,
			Notes:     task.Notes,
			Status:    "open",
			Due:       task.Due,
			CreatedAt: at,
			UpdatedAt: at,
		}
		if p.Rev, err = tx.NextRev(); err != nil {
			return err
		}
		if err := tx.PutProject(p); err != nil {
			return err
		}

		list, err := tx.AttachmentsOf(store.OwnerTask, task.ID)
		if err != nil {
			return err
		}
		for _, att := range list {
			att.TaskID = nil
			att.ProjectID = &p.ID
			att.UpdatedAt = max(at, att.UpdatedAt)
			if att.Rev, err = tx.NextRev(); err != nil {
				return err
			}
			if err := tx.PutAttachment(att); err != nil {
				return err
			}
		}

		task.DeletedAt = &at
		task.UpdatedAt = max(at, task.UpdatedAt)
		if task.Rev, err = tx.NextRev(); err != nil {
			return err
		}
		if err := tx.PutTask(task); err != nil {
			return err
		}

		res = Promoted{Project: p, RemovedTaskID: task.ID}
		return nil
	})
	return res, err
}
