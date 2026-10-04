package syncer

import (
	"context"
	"crypto/subtle"

	"homebase/internal/recur"
	"homebase/internal/review"
	"homebase/internal/store"
)

func liveRows(c store.Changes) ([]store.Goal, []store.Project, []store.Task) {
	var goals []store.Goal
	var projects []store.Project
	var tasks []store.Task
	for _, g := range c.Goals {
		if g.DeletedAt == nil {
			goals = append(goals, g)
		}
	}
	for _, p := range c.Projects {
		if p.DeletedAt == nil {
			projects = append(projects, p)
		}
	}
	for _, t := range c.Tasks {
		if t.DeletedAt == nil {
			tasks = append(tasks, t)
		}
	}
	return goals, projects, tasks
}

// Review builds this week's review for the user's zone and week start.
func (s *Syncer) Review(ctx context.Context) (review.Summary, error) {
	var out review.Summary
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		st, err := tx.Settings()
		if err != nil {
			return err
		}
		all, _, _, err := tx.ChangesSince(0, 0)
		if err != nil {
			return err
		}
		goals, projects, tasks := liveRows(all)
		out = review.Compute(review.Input{
			Now: s.now(), Loc: location(st.TZ), WeekStart: st.WeekStart,
			Goals: goals, Projects: projects, Tasks: tasks,
		})
		out.Completed, err = tx.ReviewDone(out.WeekStart)
		return err
	})
	return out, err
}

// Decision is one choice made in the review.
type Decision struct {
	Kind   string `json:"kind"`
	ID     string `json:"id"`
	Action string `json:"action"`
}

// CompleteReview applies the decisions and records the week as reviewed.
// keep touches the row so its quiet clock starts again; pause and drop set
// the status. Nothing is deleted. Rows gone since the review was loaded
// are skipped.
func (s *Syncer) CompleteReview(ctx context.Context, weekStart string, decisions []Decision) (int64, error) {
	if !recur.Valid(weekStart) {
		return 0, invalid("weekStart must be a YYYY-MM-DD date.")
	}
	if len(decisions) > 500 {
		return 0, invalid("Too many decisions.")
	}
	for _, d := range decisions {
		if !oneOf(d.Kind, "project", "goal") || !oneOf(d.Action, "keep", "pause", "drop") || !ValidID(d.ID) {
			return 0, invalid("Each decision needs kind project or goal, a ULID id, and action keep, pause or drop.")
		}
	}

	var rev int64
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		at := s.now().UnixMilli()
		status := map[string]string{"pause": "paused", "drop": "dropped"}
		for _, d := range decisions {
			var err error
			if d.Kind == "project" {
				p, gerr := tx.Project(d.ID)
				if gerr != nil || p.DeletedAt != nil {
					continue
				}
				if st, ok := status[d.Action]; ok {
					p.Status = st
				}
				p.UpdatedAt = max(at, p.UpdatedAt)
				if p.Rev, err = tx.NextRev(); err != nil {
					return err
				}
				err = tx.PutProject(p)
			} else {
				g, gerr := tx.Goal(d.ID)
				if gerr != nil || g.DeletedAt != nil {
					continue
				}
				if st, ok := status[d.Action]; ok {
					g.Status = st
				}
				g.UpdatedAt = max(at, g.UpdatedAt)
				if g.Rev, err = tx.NextRev(); err != nil {
					return err
				}
				err = tx.PutGoal(g)
			}
			if err != nil {
				return err
			}
		}
		if err := tx.LogReview(weekStart, at); err != nil {
			return err
		}
		var err error
		rev, err = tx.Rev()
		return err
	})
	return rev, err
}

// CalendarToken returns the secret part of the calendar link.
func (s *Syncer) CalendarToken(ctx context.Context) (string, error) {
	var token string
	err := s.store.Tx(ctx, func(tx store.Tx) error {
		st, err := tx.Settings()
		token = st.CalendarToken
		return err
	})
	return token, err
}

// RotateCalendar replaces the calendar token; the old link stops at once.
func (s *Syncer) RotateCalendar(ctx context.Context) (string, error) {
	token := NewSecret()
	err := s.store.Tx(ctx, func(tx store.Tx) error { return tx.SetCalendarToken(token) })
	return token, err
}

// CalendarTasks returns the live tasks and the user's zone when token is
// the current calendar token, compared in constant time. ok is false for a
// wrong token.
func (s *Syncer) CalendarTasks(ctx context.Context, token string) (tasks []store.Task, tz string, ok bool, err error) {
	err = s.store.Tx(ctx, func(tx store.Tx) error {
		st, err := tx.Settings()
		if err != nil {
			return err
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(st.CalendarToken)) != 1 {
			return nil
		}
		ok, tz = true, st.TZ
		all, _, _, err := tx.ChangesSince(0, 0)
		if err != nil {
			return err
		}
		_, _, tasks = liveRows(all)
		return nil
	})
	return tasks, tz, ok, err
}
