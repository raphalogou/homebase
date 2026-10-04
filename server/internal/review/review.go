// Package review computes the weekly review of docs/SPEC.md section 5:
// what got done this week, what has gone quiet, and which goals saw nothing.
package review

import (
	"sort"
	"time"

	"homebase/internal/store"
)

// QuietAfter is how long without activity makes a project or goal quiet.
const QuietAfter = 21 * 24 * time.Hour

// Input is everything the review looks at, already filtered to live rows.
type Input struct {
	Now       time.Time
	Loc       *time.Location
	WeekStart int // 1 Monday, 0 Sunday
	Goals     []store.Goal
	Projects  []store.Project
	Tasks     []store.Task
}

// GoalCount is how many tasks were done for one goal this week.
type GoalCount struct {
	GoalID string `json:"goalId"`
	Title  string `json:"title"`
	Count  int    `json:"count"`
}

// Quiet is a project or goal with no activity for QuietAfter.
type Quiet struct {
	Kind         string `json:"kind"` // "project" or "goal"
	ID           string `json:"id"`
	Title        string `json:"title"`
	LastActivity int64  `json:"lastActivity"`
}

// GoalRef names a goal.
type GoalRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// Summary is the answer to GET /api/review.
type Summary struct {
	WeekStart        string      `json:"weekStart"`
	DoneCount        int         `json:"doneCount"`
	DoneByGoal       []GoalCount `json:"doneByGoal"`
	Quiet            []Quiet     `json:"quiet"`
	GoalsWithNothing []GoalRef   `json:"goalsWithNothing"`
	// Completed says whether this week's review was finished; Today shows
	// its link from Friday until it is.
	Completed bool `json:"completed"`
}

const day = "2006-01-02"

// WeekStartOf returns the first day of now's week in loc.
func WeekStartOf(now time.Time, loc *time.Location, weekStart int) string {
	local := now.In(loc)
	back := (int(local.Weekday()) - weekStart + 7) % 7
	y, m, d := local.Date()
	return time.Date(y, m, d-back, 0, 0, 0, 0, time.UTC).Format(day)
}

// Compute builds the review. Completed is left for the caller.
func Compute(in Input) Summary {
	ws := WeekStartOf(in.Now, in.Loc, in.WeekStart)
	wsDate, _ := time.Parse(day, ws)
	we := wsDate.AddDate(0, 0, 7).Format(day)
	cutoff := in.Now.Add(-QuietAfter).UnixMilli()
	cutoffDay := in.Now.In(in.Loc).Add(-QuietAfter).Format(day)

	projectGoal := map[string]*string{}
	for _, p := range in.Projects {
		projectGoal[p.ID] = p.GoalID
	}
	goalOf := func(t store.Task) string {
		if t.GoalID != nil {
			return *t.GoalID
		}
		if t.ProjectID != nil {
			if g := projectGoal[*t.ProjectID]; g != nil {
				return *g
			}
		}
		return ""
	}

	s := Summary{WeekStart: ws, DoneByGoal: []GoalCount{}, Quiet: []Quiet{}, GoalsWithNothing: []GoalRef{}}

	// Done this week, overall and per goal.
	perGoal := map[string]int{}
	for _, t := range in.Tasks {
		if t.Status != "done" || t.DoneAt == nil {
			continue
		}
		d := time.UnixMilli(*t.DoneAt).In(in.Loc).Format(day)
		if d < ws || d >= we {
			continue
		}
		s.DoneCount++
		if g := goalOf(t); g != "" {
			perGoal[g]++
		}
	}

	// Activity per project and per goal.
	projectActive := map[string]int64{}
	projectOpenTasks := map[string]int{}
	goalActive := map[string]int64{}
	for _, t := range in.Tasks {
		last := max(t.CreatedAt, t.UpdatedAt)
		if t.DoneAt != nil {
			last = max(last, *t.DoneAt)
		}
		if t.ProjectID != nil {
			projectActive[*t.ProjectID] = max(projectActive[*t.ProjectID], last)
			if t.Status == "open" {
				projectOpenTasks[*t.ProjectID]++
			}
		}
		// A goal counts as active when one of its tasks was planned or done
		// recently, not merely edited.
		if g := goalOf(t); g != "" {
			var act int64
			if t.DoneAt != nil {
				act = *t.DoneAt
			}
			if t.PlannedOn != nil && *t.PlannedOn >= cutoffDay {
				act = max(act, in.Now.UnixMilli())
			}
			goalActive[g] = max(goalActive[g], act)
		}
	}

	for _, p := range in.Projects {
		if p.Status != "open" || projectOpenTasks[p.ID] == 0 {
			continue
		}
		last := max(p.UpdatedAt, projectActive[p.ID])
		if last < cutoff {
			s.Quiet = append(s.Quiet, Quiet{Kind: "project", ID: p.ID, Title: p.Title, LastActivity: last})
		}
	}
	for _, g := range in.Goals {
		if g.Status != "open" {
			continue
		}
		last := max(g.UpdatedAt, goalActive[g.ID])
		if last < cutoff {
			s.Quiet = append(s.Quiet, Quiet{Kind: "goal", ID: g.ID, Title: g.Title, LastActivity: last})
		}
		if n := perGoal[g.ID]; n > 0 {
			s.DoneByGoal = append(s.DoneByGoal, GoalCount{GoalID: g.ID, Title: g.Title, Count: n})
		} else {
			s.GoalsWithNothing = append(s.GoalsWithNothing, GoalRef{ID: g.ID, Title: g.Title})
		}
	}

	sort.SliceStable(s.DoneByGoal, func(i, j int) bool { return s.DoneByGoal[i].Count > s.DoneByGoal[j].Count })
	sort.SliceStable(s.Quiet, func(i, j int) bool { return s.Quiet[i].LastActivity < s.Quiet[j].LastActivity })
	return s
}
