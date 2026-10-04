package sched

import (
	"fmt"
	"strings"
	"time"

	"homebase/internal/store"
)

// Message is one notification, written from current data
// (docs/SPEC.md section 6). Every reminder opens Today.
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	// Tag makes a later reminder of the same slot replace an unread one.
	Tag string `json:"tag"`
	// Sync asks the open app to sync, so it shows what the message says.
	Sync bool `json:"sync"`
}

// Grace is how late a slot may still fire, for a server that was briefly down.
const Grace = 90 * time.Minute

// slotDue reports whether a slot set for atLocal ("HH:MM") should fire at
// local time now: at or after the time and less than Grace later.
func slotDue(now time.Time, atLocal string) bool {
	at, err := time.Parse("15:04", atLocal)
	if err != nil {
		return false
	}
	y, m, d := now.Date()
	when := time.Date(y, m, d, at.Hour(), at.Minute(), 0, 0, now.Location())
	return !now.Before(when) && now.Sub(when) < Grace
}

// BuildMessage writes the text for a reminder of kind on localDate.
// goals are the open goals in sort_key order; planned are the day's
// non-dropped tasks in rank order.
func BuildMessage(kind string, slot int, localDate string, goals []store.Goal, planned []store.Task) Message {
	m := Message{URL: "/", Tag: fmt.Sprintf("reminder-%d", slot), Sync: true}

	var open []store.Task
	done := 0
	for _, t := range planned {
		if t.Status == "done" {
			done++
		} else {
			open = append(open, t)
		}
	}
	progress := fmt.Sprintf("%d of %d done.", done, len(planned))

	switch kind {
	case "focus":
		m.Title = "Today"
		if len(goals) > 0 {
			m.Title = goals[dayNumber(localDate)%len(goals)].Title
		}
		switch {
		case len(open) > 0:
			m.Body = "Today: " + titles(open) + "."
		case len(planned) > 0:
			m.Body = "All done for today."
		default:
			m.Body = "Choose your three for today."
		}
	case "checkin":
		m.Title = "Midday check"
		switch {
		case len(planned) == 0:
			m.Body = "Nothing planned yet. Choose your three for today."
		case len(open) == 0:
			m.Body = "All done for today."
		default:
			m.Body = progress + " Next: " + open[0].Title + "."
		}
	case "wrap":
		m.Title = "Close the day"
		switch {
		case len(planned) == 0:
			m.Body = "Nothing planned today. Capture what is on your mind."
		case len(open) == 0:
			m.Body = "All done for today."
		default:
			m.Body = progress + " Move the rest or let it go."
		}
	default:
		m.Title = "Homebase"
		m.Body = "Open Today."
	}
	return m
}

func titles(tasks []store.Task) string {
	list := make([]string, len(tasks))
	for i, t := range tasks {
		list[i] = strings.TrimRight(t.Title, ".")
	}
	return strings.Join(list, "; ")
}

// dayNumber counts days since 1970-01-01, so the focus goal turns over
// once a day and in the same order on every device.
func dayNumber(localDate string) int {
	d, err := time.Parse("2006-01-02", localDate)
	if err != nil {
		return 0
	}
	return int(d.Unix() / 86400)
}
