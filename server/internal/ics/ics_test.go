package ics

import (
	"strings"
	"testing"
	"time"

	"homebase/internal/store"
)

func p[T any](v T) *T { return &v }

func TestFeedEvents(t *testing.T) {
	stamp := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC).UnixMilli()
	task := func(id, status string, due, planned *string) store.Task {
		return store.Task{ID: id, Title: "Task " + id, Status: status, Due: due, PlannedOn: planned, UpdatedAt: stamp}
	}
	tasks := []store.Task{
		task("A", "open", p("2026-03-06"), p("2026-03-05")), // both: two events
		task("B", "open", p("2026-02-20"), nil),             // 12 days ago: outside
		task("C", "open", p("2026-02-25"), nil),             // 7 days ago: inside
		task("D", "done", p("2026-03-06"), nil),             // done: left out
		task("E", "open", p("2026-08-31"), nil),             // 180 days on: inside
		task("F", "open", p("2026-09-01"), nil),             // 181 days on: outside
		task("G", "open", nil, nil),                         // undated: left out
	}
	body, etag := Feed(tasks, "2026-03-04")
	s := string(body)

	for _, want := range []string{
		"UID:task-A-due@homebase\r\n",
		"UID:task-A-plan@homebase\r\n",
		"SUMMARY:Due: Task A\r\n",
		"SUMMARY:Plan: Task A\r\n",
		"DTSTART;VALUE=DATE:20260306\r\nDTEND;VALUE=DATE:20260307\r\n",
		"DTSTAMP:20260301T093000Z\r\n",
		"UID:task-C-due@homebase",
		"UID:task-E-due@homebase",
		"REFRESH-INTERVAL;VALUE=DURATION:PT1H\r\n",
		"X-PUBLISHED-TTL:PT1H\r\n",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("feed lacks %q", want)
		}
	}
	for _, unwanted := range []string{"task-B-", "task-D-", "task-F-", "task-G-"} {
		if strings.Contains(s, unwanted) {
			t.Errorf("feed has %q", unwanted)
		}
	}
	if n := strings.Count(s, "BEGIN:VEVENT"); n != 4 {
		t.Errorf("events = %d, want 4", n)
	}
	if !strings.HasPrefix(s, "BEGIN:VCALENDAR\r\n") || !strings.HasSuffix(s, "END:VCALENDAR\r\n") {
		t.Error("feed is not a calendar")
	}
	if strings.Contains(strings.ReplaceAll(s, "\r\n", ""), "\n") {
		t.Error("bare LF in feed")
	}

	again, etag2 := Feed(tasks, "2026-03-04")
	if string(again) != s || etag2 != etag {
		t.Error("the same data gave a different feed or ETag")
	}
	tasks[0].Title = "Renamed"
	if _, etag3 := Feed(tasks, "2026-03-04"); etag3 == etag {
		t.Error("ETag did not change with the content")
	}
}

func TestEscape(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Buy milk, eggs; bread", `Buy milk\, eggs\; bread`},
		{`C:\temp`, `C:\\temp`},
		{"two\nlines", `two\nlines`},
	}
	for _, tt := range tests {
		if got := escape(tt.in); got != tt.want {
			t.Errorf("escape(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFolding(t *testing.T) {
	body, _ := Feed([]store.Task{{ID: "X", Title: strings.Repeat("é", 60), Status: "open", Due: p("2026-03-04")}}, "2026-03-04")
	for _, l := range strings.Split(string(body), "\r\n") {
		if len(l) > 75 {
			t.Errorf("line of %d octets: %q", len(l), l)
		}
	}
	unfolded := strings.ReplaceAll(string(body), "\r\n ", "")
	if !strings.Contains(unfolded, "SUMMARY:Due: "+strings.Repeat("é", 60)) {
		t.Error("unfolding does not give the title back whole")
	}
}
