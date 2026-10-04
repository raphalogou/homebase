// Package ics writes the private calendar feed (docs/SPEC.md section 6):
// all-day events for due and planned tasks, readable by Google Calendar.
package ics

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"homebase/internal/store"
)

const (
	day      = "2006-01-02"
	pastDays = 7
	nextDays = 180
)

type event struct {
	uid     string
	date    string
	summary string
	stamp   int64
}

// Feed returns the calendar for the open tasks with a due or planned day
// between 7 days before today and 180 days after, and an ETag for it.
// today is the user's local day.
func Feed(tasks []store.Task, today string) (body []byte, etag string) {
	t0, _ := time.Parse(day, today)
	from := t0.AddDate(0, 0, -pastDays).Format(day)
	to := t0.AddDate(0, 0, nextDays).Format(day)
	in := func(d *string) bool { return d != nil && *d >= from && *d <= to }

	var events []event
	for _, t := range tasks {
		if t.Status != "open" || t.DeletedAt != nil {
			continue
		}
		if in(t.Due) {
			events = append(events, event{uid: "task-" + t.ID + "-due@homebase", date: *t.Due, summary: "Due: " + t.Title, stamp: t.UpdatedAt})
		}
		if in(t.PlannedOn) {
			events = append(events, event{uid: "task-" + t.ID + "-plan@homebase", date: *t.PlannedOn, summary: "Plan: " + t.Title, stamp: t.UpdatedAt})
		}
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].date != events[j].date {
			return events[i].date < events[j].date
		}
		return events[i].uid < events[j].uid
	})

	var b bytes.Buffer
	line := func(s string) { writeFolded(&b, s) }
	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//Homebase//Homebase//EN")
	line("CALSCALE:GREGORIAN")
	line("METHOD:PUBLISH")
	line("X-WR-CALNAME:Homebase")
	// Hints only: Google Calendar polls on its own schedule, every several hours.
	line("REFRESH-INTERVAL;VALUE=DURATION:PT1H")
	line("X-PUBLISHED-TTL:PT1H")
	for _, e := range events {
		d, _ := time.Parse(day, e.date)
		line("BEGIN:VEVENT")
		line("UID:" + e.uid)
		line("DTSTAMP:" + time.UnixMilli(e.stamp).UTC().Format("20060102T150405Z"))
		line("DTSTART;VALUE=DATE:" + d.Format("20060102"))
		line("DTEND;VALUE=DATE:" + d.AddDate(0, 0, 1).Format("20060102"))
		line("SUMMARY:" + escape(e.summary))
		line("TRANSP:TRANSPARENT")
		line("END:VEVENT")
	}
	line("END:VCALENDAR")

	sum := sha256.Sum256(b.Bytes())
	return b.Bytes(), `"` + hex.EncodeToString(sum[:16]) + `"`
}

// escape follows RFC 5545 section 3.3.11 for TEXT values.
func escape(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`, "\r", `\n`).Replace(s)
}

// writeFolded writes one content line, folded at 75 octets without
// splitting a UTF-8 character, and ends it with CRLF (RFC 5545 3.1).
func writeFolded(b *bytes.Buffer, s string) {
	limit := 75
	for len(s) > limit {
		cut := limit
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		b.WriteString(s[:cut])
		b.WriteString("\r\n ")
		s = s[cut:]
		limit = 74 // the leading space of a continuation line counts
	}
	b.WriteString(s)
	b.WriteString("\r\n")
}
