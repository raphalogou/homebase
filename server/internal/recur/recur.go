// Package recur computes the next due date of a repeating task.
//
// Dates are local calendar days ("YYYY-MM-DD"). They are parsed as UTC
// midnight only to do arithmetic; no time zone is involved.
package recur

import (
	"fmt"
	"time"
)

const layout = "2006-01-02"

// Rule mirrors a repeats row.
type Rule struct {
	Freq     string // day, week, month
	Every    int
	Weekdays int    // bit mask, Mon=1 ... Sun=64; 0 means none
	Mode     string // fixed, after_done
	Until    string // "" means no end
}

// Next returns the due date of the successor of a task that had due date
// oldDue ("" if none) and was completed or dropped on doneOn. today is the
// current local day. ok is false when the chain has ended.
func Next(r Rule, oldDue, doneOn, today string) (next string, ok bool, err error) {
	if r.Every < 1 {
		return "", false, fmt.Errorf("every must be at least 1, got %d", r.Every)
	}
	if r.Until != "" && r.Until < today {
		return "", false, nil
	}

	var d time.Time
	switch r.Mode {
	case "after_done":
		base, err := parse(doneOn)
		if err != nil {
			return "", false, err
		}
		d, err = afterDone(r, base)
		if err != nil {
			return "", false, err
		}
	case "fixed":
		from := oldDue
		if from == "" {
			from = doneOn
		}
		base, err := parse(from)
		if err != nil {
			return "", false, err
		}
		d, err = fixed(r, base)
		if err != nil {
			return "", false, err
		}
	default:
		return "", false, fmt.Errorf("unknown mode %q", r.Mode)
	}

	next = d.Format(layout)
	if r.Until != "" && next > r.Until {
		return "", false, nil
	}
	return next, true, nil
}

// Valid reports whether s is a real calendar date in YYYY-MM-DD form.
func Valid(s string) bool {
	_, err := parse(s)
	return err == nil
}

// Date returns the local calendar day of t.
func Date(t time.Time) string {
	return t.Format(layout)
}

func parse(s string) (time.Time, error) {
	d, err := time.Parse(layout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("date %q: %w", s, err)
	}
	return d, nil
}

func afterDone(r Rule, base time.Time) (time.Time, error) {
	switch r.Freq {
	case "day":
		return base.AddDate(0, 0, r.Every), nil
	case "week":
		return base.AddDate(0, 0, 7*r.Every), nil
	case "month":
		return addMonths(base, r.Every, false), nil
	default:
		return time.Time{}, fmt.Errorf("unknown freq %q", r.Freq)
	}
}

func fixed(r Rule, base time.Time) (time.Time, error) {
	switch r.Freq {
	case "day":
		return base.AddDate(0, 0, r.Every), nil
	case "week":
		if r.Weekdays == 0 {
			return base.AddDate(0, 0, 7*r.Every), nil
		}
		return nextWeekday(base, r.Every, r.Weekdays), nil
	case "month":
		return addMonths(base, r.Every, isLastDay(base)), nil
	default:
		return time.Time{}, fmt.Errorf("unknown freq %q", r.Freq)
	}
}

// nextWeekday finds the first day after base whose weekday is in mask and
// whose week (Monday to Sunday) is a multiple of every weeks from base's.
func nextWeekday(base time.Time, every, mask int) time.Time {
	baseWeek := mondayOf(base)
	for i := 1; i <= 7*(every+1); i++ {
		d := base.AddDate(0, 0, i)
		weeks := int(mondayOf(d).Sub(baseWeek).Hours() / (24 * 7))
		if mask&weekdayBit(d) != 0 && weeks%every == 0 {
			return d
		}
	}
	// Unreachable for a mask in 1..127; fall back to the plain interval.
	return base.AddDate(0, 0, 7*every)
}

func weekdayBit(d time.Time) int {
	// Go counts Sunday as 0; the mask counts Monday as bit 0.
	return 1 << ((int(d.Weekday()) + 6) % 7)
}

func mondayOf(d time.Time) time.Time {
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// addMonths moves by n months and clamps to the end of a shorter month
// (31 January plus one month is 28 or 29 February). With stickToEnd, a date
// on the last day of its month lands on the last day of the target month, so
// a fixed rule anchored on the 31st does not drift to the 28th for good.
func addMonths(d time.Time, n int, stickToEnd bool) time.Time {
	y, m, day := d.Date()
	first := time.Date(y, m+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	last := daysIn(first)
	if stickToEnd || day > last {
		day = last
	}
	return first.AddDate(0, 0, day-1)
}

func daysIn(firstOfMonth time.Time) int {
	return firstOfMonth.AddDate(0, 1, -1).Day()
}

func isLastDay(d time.Time) bool {
	return d.AddDate(0, 0, 1).Day() == 1
}
