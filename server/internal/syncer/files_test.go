package syncer

import (
	"strings"
	"testing"
	"time"

	"homebase/internal/files"
)

func TestOrphanFiles(t *testing.T) {
	sha := func(c byte) string {
		b := make([]byte, 64)
		for i := range b {
			b[i] = c
		}
		return string(b)
	}
	tests := []struct {
		name   string
		setup  func(e *env) // runs at e.now, then the clock moves 8 days on
		orphan bool
	}{
		{"used file stays", func(e *env) {
			attach(e, sha('a'))
		}, false},
		{"deleted long ago goes", func(e *env) {
			id := attach(e, sha('b'))
			e.mustApply(del("attachments", id, e.at(time.Second), ""))
		}, true},
		{"deleted recently stays", func(e *env) {
			id := attach(e, sha('c'))
			e.now = e.now.Add(7 * 24 * time.Hour)
			e.mustApply(del("attachments", id, e.at(0), ""))
			e.now = e.now.Add(-7 * 24 * time.Hour)
		}, false},
		{"used by a second attachment stays", func(e *env) {
			id := attach(e, sha('d'))
			attach(e, sha('d'))
			e.mustApply(del("attachments", id, e.at(time.Second), ""))
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			e.mustApply(upsert("projects", id(1), e.at(0), projRow("p")))
			tt.setup(e)
			e.now = e.now.Add(8 * 24 * time.Hour)
			list, err := e.s.OrphanFiles(e.ctx)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(list) == 1; got != tt.orphan {
				t.Errorf("orphans = %v, want orphan=%v", list, tt.orphan)
			}
		})
	}
}

func attach(e *env, sha string) string {
	e.t.Helper()
	a, err := e.s.AttachFile(e.ctx, "project", id(1), "report.pdf", files.Saved{SHA: sha, Size: 3, Mime: "application/pdf"})
	if err != nil {
		e.t.Fatal(err)
	}
	return a.ID
}

func TestAttachFileCleansName(t *testing.T) {
	e := newEnv(t)
	e.mustApply(upsert("projects", id(1), e.at(0), projRow("p")))
	tests := []struct{ in, want string }{
		{"../../etc/passwd", "passwd"},
		{`C:\Users\me\plan.pdf`, "plan.pdf"},
		{"line\r\nbreak.txt", "linebreak.txt"},
		{"   ", "File"},
	}
	for _, tt := range tests {
		a, err := e.s.AttachFile(e.ctx, "project", id(1), tt.in, files.Saved{SHA: strings.Repeat("ab", 32), Size: 1, Mime: "text/plain"})
		if err != nil {
			t.Fatal(err)
		}
		if a.Name != tt.want {
			t.Errorf("name %q -> %q, want %q", tt.in, a.Name, tt.want)
		}
	}
}
