package syncer

import (
	"math"
	"net/url"
	"strings"
	"unicode/utf8"

	"homebase/internal/recur"
	"homebase/internal/store"
)

// Limits from docs/SPEC.md section 1. The database checks them too; these
// checks give the client a clear reason instead of a constraint error.
const (
	MaxTitle = 300
	MaxNotes = 20000
	MaxName  = 300
)

func validTitle(s string) bool {
	n := utf8.RuneCountInString(s)
	return utf8.ValidString(s) && n >= 1 && n <= MaxTitle && strings.TrimSpace(s) != ""
}

func validNotes(s string, limit int) bool {
	return utf8.ValidString(s) && utf8.RuneCountInString(s) <= limit
}

func validDate(d *string) bool { return d == nil || recur.Valid(*d) }

func validRef(id *string) bool { return id == nil || ValidID(*id) }

func oneOf(s string, options ...string) bool {
	for _, o := range options {
		if s == o {
			return true
		}
	}
	return false
}

func validateGoal(g store.Goal) error {
	switch {
	case !validTitle(g.Title):
		return invalid("Title must be 1 to 300 characters.")
	case !validNotes(g.Notes, MaxNotes):
		return invalid("Notes must be at most 20,000 characters.")
	case !oneOf(g.Status, "open", "paused", "done", "dropped"):
		return invalid("Goal status must be open, paused, done or dropped.")
	case !validDate(g.TargetDate):
		return invalid("targetDate must be a YYYY-MM-DD date.")
	case math.IsNaN(g.SortKey) || math.IsInf(g.SortKey, 0):
		return invalid("sortKey must be a finite number.")
	}
	return nil
}

func validateProject(p store.Project) error {
	switch {
	case !validTitle(p.Title):
		return invalid("Title must be 1 to 300 characters.")
	case !validNotes(p.Notes, MaxNotes):
		return invalid("Notes must be at most 20,000 characters.")
	case !oneOf(p.Status, "open", "paused", "done", "dropped"):
		return invalid("Project status must be open, paused, done or dropped.")
	case !validDate(p.Due):
		return invalid("due must be a YYYY-MM-DD date.")
	case !validRef(p.GoalID):
		return invalid("goalId must be a ULID.")
	}
	return nil
}

func validateTask(t store.Task) error {
	switch {
	case !validTitle(t.Title):
		return invalid("Title must be 1 to 300 characters.")
	case !validNotes(t.Notes, MaxNotes):
		return invalid("Notes must be at most 20,000 characters.")
	case !oneOf(t.Status, "open", "done", "dropped"):
		return invalid("Task status must be open, done or dropped.")
	case !validDate(t.Due):
		return invalid("due must be a YYYY-MM-DD date.")
	case !validDate(t.PlannedOn):
		return invalid("plannedOn must be a YYYY-MM-DD date.")
	case t.PlanRank != nil && (math.IsNaN(*t.PlanRank) || math.IsInf(*t.PlanRank, 0)):
		return invalid("planRank must be a finite number.")
	case !validRef(t.ProjectID) || !validRef(t.GoalID) || !validRef(t.RepeatID):
		return invalid("projectId, goalId and repeatId must be ULIDs.")
	case t.ProjectID != nil && t.GoalID != nil:
		return invalid("A task belongs to a project or directly to a goal, not both.")
	}
	return nil
}

func validateRepeat(r store.Repeat) error {
	switch {
	case !oneOf(r.Freq, "day", "week", "month"):
		return invalid("freq must be day, week or month.")
	case r.Every < 1 || r.Every > 365:
		return invalid("every must be between 1 and 365.")
	case r.Weekdays != nil && (*r.Weekdays < 1 || *r.Weekdays > 127):
		return invalid("weekdays must be a mask between 1 and 127.")
	case !oneOf(r.Mode, "fixed", "after_done"):
		return invalid("mode must be fixed or after_done.")
	case !validDate(r.Until):
		return invalid("until must be a YYYY-MM-DD date.")
	}
	return nil
}

func validateAttachment(t store.Attachment) error {
	owners := 0
	for _, id := range []*string{t.GoalID, t.ProjectID, t.TaskID} {
		if id != nil {
			owners++
		}
	}
	switch {
	case owners != 1:
		return invalid("An attachment belongs to exactly one goal, project or task.")
	case !validRef(t.GoalID) || !validRef(t.ProjectID) || !validRef(t.TaskID):
		return invalid("Owner ids must be ULIDs.")
	case !validNotes(t.Name, MaxName):
		return invalid("Name must be at most 300 characters.")
	case t.Body != nil && !validNotes(*t.Body, MaxNotes):
		return invalid("A note must be at most 20,000 characters.")
	}

	switch t.Kind {
	case "link":
		if t.URL == nil || !validURL(*t.URL) {
			return invalid("A link needs an http, https or mailto URL.")
		}
	case "note":
		if t.Body == nil {
			return invalid("A note needs a body.")
		}
	case "file":
		if t.FileSHA == nil || !validSHA(*t.FileSHA) {
			return invalid("A file needs the SHA-256 of an uploaded file.")
		}
	default:
		return invalid("kind must be link, note or file.")
	}
	return nil
}

func validURL(s string) bool {
	if len(s) > 2048 {
		return false
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	switch u.Scheme {
	case "http", "https":
		return u.Host != ""
	case "mailto":
		return u.Opaque != ""
	}
	return false
}

func validSHA(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
