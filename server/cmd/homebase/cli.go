package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const usage = `Homebase, a calm personal planner: the server and the web app in one binary.

Usage:
  homebase <command> [arguments]

Commands:
  serve              Start the server (Ctrl-C stops it)
  hash-passphrase    Read a passphrase twice and print HOMEBASE_PASSPHRASE_HASH
  backup <dir>       Copy everyone's database and files, and the push key, into <dir>
  version            Print the version and build commit
  help               Show this help

Settings come from environment variables (defaults in brackets):
  HOMEBASE_DATA              Data folder [./data]
  HOMEBASE_ADDR              Listen address [:8080]
  HOMEBASE_BASE_URL          Public HTTPS address, used in the calendar link
  HOMEBASE_VAPID_SUBJECT     Contact for push services [mailto:admin@localhost]
  HOMEBASE_PASSPHRASE_HASH   Optional: creates the account on first start
  HOMEBASE_BACKUP_DIR        Where backups go once on in Settings [<data>/backups]
  HOMEBASE_BACKUP_INTERVAL   How often, like 7d or 12h [1d]

Examples:
  homebase serve
  HOMEBASE_DATA=/srv/homebase HOMEBASE_ADDR=127.0.0.1:8080 homebase serve
  homebase backup /var/backups/homebase
`

// usageError is a mistake in the command line itself; main adds a pointer
// to the help and exits with 2, as most commands do.
type usageError string

func (e usageError) Error() string { return string(e) }

// ANSI colours, used only on a terminal and never with NO_COLOR set.
const (
	reset  = "\x1b[0m"
	bold   = "\x1b[1m"
	dim    = "\x1b[2m"
	red    = "\x1b[31m"
	green  = "\x1b[32m"
	yellow = "\x1b[33m"
	cyan   = "\x1b[36m"
)

// palette returns its colour, or "" when w is not a colour terminal.
type palette bool

func (p palette) c(code string) string {
	if p {
		return code
	}
	return ""
}

func colorFor(w io.Writer, getenv func(string) string) palette {
	f, ok := w.(*os.File)
	return palette(ok && isTerminal(f) && getenv("NO_COLOR") == "" && getenv("TERM") != "dumb")
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// newLogger writes short coloured lines on a terminal ("14:02:11 INFO
// listening addr=:8080") and slog's plain key=value text elsewhere, where
// Docker, journald or a file collect it.
func newLogger(w io.Writer, p palette) *slog.Logger {
	if !p {
		return slog.New(slog.NewTextHandler(w, nil))
	}
	return slog.New(&prettyHandler{w: w, mu: &sync.Mutex{}})
}

type prettyHandler struct {
	w     io.Writer
	mu    *sync.Mutex
	attrs string // from WithAttrs, already formatted
	group string
}

func (h *prettyHandler) Enabled(_ context.Context, l slog.Level) bool { return l >= slog.LevelInfo }

func (h *prettyHandler) Handle(_ context.Context, r slog.Record) error {
	level := map[slog.Level]string{slog.LevelError: red, slog.LevelWarn: yellow, slog.LevelInfo: green}[r.Level]
	var b strings.Builder
	fmt.Fprintf(&b, "%s%s%s %s%-5s%s %s", dim, r.Time.Format("15:04:05"), reset, level, r.Level, reset, r.Message)
	b.WriteString(h.attrs)
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&b, h.group, a)
		return true
	})
	b.WriteByte('\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, b.String())
	return err
}

func (h *prettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	var b strings.Builder
	for _, a := range attrs {
		writeAttr(&b, h.group, a)
	}
	return &prettyHandler{w: h.w, mu: h.mu, attrs: h.attrs + b.String(), group: h.group}
}

func (h *prettyHandler) WithGroup(name string) slog.Handler {
	return &prettyHandler{w: h.w, mu: h.mu, attrs: h.attrs, group: joinKey(h.group, name)}
}

func writeAttr(b *strings.Builder, group string, a slog.Attr) {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return
	}
	key := joinKey(group, a.Key)
	if a.Value.Kind() == slog.KindGroup {
		for _, g := range a.Value.Group() {
			writeAttr(b, key, g)
		}
		return
	}
	val := a.Value.String()
	if val == "" || strings.ContainsAny(val, " \"=\n") {
		val = strconv.Quote(val)
	}
	color := ""
	if a.Key == "err" {
		color = red
	}
	fmt.Fprintf(b, " %s%s=%s%s%s%s", dim, key, reset, color, val, reset)
}

func joinKey(group, key string) string {
	if group == "" {
		return key
	}
	return group + "." + key
}

// summary is what serve prints once it listens: where to open Homebase and
// what it is set to do, so a first start needs no reading of logs.
type summary struct {
	Addr, BaseURL, DataDir, BackupDir string
	BackupInterval                    time.Duration
	People                            int
}

func printSummary(w io.Writer, p palette, s summary) {
	line := func(label, value string) {
		_, _ = fmt.Fprintf(w, "  %s%-9s%s %s\n", p.c(dim), label, p.c(reset), value)
	}
	_, _ = fmt.Fprintf(w, "\n  %sHomebase %s%s %s(%s)%s\n\n", p.c(bold), version, p.c(reset), p.c(dim), commit, p.c(reset))
	line("Local", p.c(cyan)+localURL(s.Addr)+p.c(reset))
	if s.BaseURL != "" {
		line("Public", p.c(cyan)+s.BaseURL+p.c(reset))
	} else {
		line("Public", "not set "+p.c(dim)+"(HOMEBASE_BASE_URL; the calendar link uses the address it is opened at)"+p.c(reset))
	}
	line("Data", s.DataDir)
	line("Backups", fmt.Sprintf("every %s to %s, for each person who turns them on in Settings", every(s.BackupInterval), s.BackupDir))
	switch s.People {
	case 0:
		line("People", p.c(yellow)+"none yet: open the address above to set up the owner"+p.c(reset))
	case 1:
		line("People", "1, the owner")
	default:
		line("People", fmt.Sprintf("%d", s.People))
	}
	_, _ = fmt.Fprintln(w)
}

// localURL turns a listen address into one a browser on this machine can
// open: ":8080" and "0.0.0.0:8080" become http://localhost:8080.
func localURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}

// every matches Settings: "day", "week", "3 days", "12h0m0s".
func every(d time.Duration) string {
	switch {
	case d == 24*time.Hour:
		return "day"
	case d == 7*24*time.Hour:
		return "week"
	case d%(24*time.Hour) == 0:
		return fmt.Sprintf("%d days", d/(24*time.Hour))
	}
	return d.String()
}
