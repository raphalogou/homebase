// Package store holds all SQL. Business rules live in the callers (syncer,
// auth), which work through the Tx interface.
package store

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a row does not exist.
var ErrNotFound = errors.New("not found")

// Store runs transactions.
type Store interface {
	// Tx runs fn in one write transaction. fn's error rolls it back.
	Tx(ctx context.Context, fn func(Tx) error) error
	Close() error
}

// Tx is every query the server needs, inside one transaction. Get methods
// return deleted rows too; callers decide what a tombstone means.
type Tx interface {
	// NextRev increments meta.rev and returns the new value. Every write to a
	// synced table takes its own revision, so revisions are unique per row.
	NextRev() (int64, error)
	Rev() (int64, error)

	// Savepoint runs fn inside a savepoint and undoes fn's writes if it fails,
	// leaving the rest of the transaction intact.
	Savepoint(fn func() error) error

	Settings() (Settings, error)
	InsertSettings(Settings) error
	SetLastRollover(date string) error

	Reminders() ([]Reminder, error)
	PutReminder(Reminder) error

	Goal(id string) (Goal, error)
	PutGoal(Goal) error
	Project(id string) (Project, error)
	PutProject(Project) error
	Repeat(id string) (Repeat, error)
	PutRepeat(Repeat) error
	Task(id string) (Task, error)
	PutTask(Task) error
	Attachment(id string) (Attachment, error)
	PutAttachment(Attachment) error

	// CountPlanned counts live, non-dropped tasks planned on date, except
	// the task with id exceptID.
	CountPlanned(date, exceptID string) (int, error)
	// OpenTasksPlannedBefore lists live open tasks with planned_on < date.
	OpenTasksPlannedBefore(date string) ([]Task, error)
	// HasOpenInstance reports whether a live open task other than exceptID
	// carries repeatID.
	HasOpenInstance(repeatID, exceptID string) (bool, error)

	// Live children, used by cascade delete and promote.
	ProjectsOfGoal(goalID string) ([]Project, error)
	TasksOfGoal(goalID string) ([]Task, error)
	TasksOfProject(projectID string) ([]Task, error)
	AttachmentsOf(kind OwnerKind, ownerID string) ([]Attachment, error)
	FileExists(sha string) (bool, error)
	// InsertFile records a stored file; recording the same file again is a no-op.
	InsertFile(f File) error
	File(sha string) (File, error)
	// OrphanFiles lists files created before cutoff that no live attachment
	// uses and no attachment changed after cutoff.
	OrphanFiles(cutoff int64) ([]string, error)

	// ChangesSince returns synced rows with rev > since, oldest first. With
	// limit > 0 it returns at most limit rows, and more reports whether rows
	// were left out; last is the rev of the last row returned (or since).
	ChangesSince(since int64, limit int) (c Changes, more bool, last int64, err error)

	UpdateSettings(tz string, weekStart int) error

	// OpenGoals lists live open goals in sort_key order.
	OpenGoals() ([]Goal, error)
	// TasksPlannedOn lists live, non-dropped tasks planned on date, in rank order.
	TasksPlannedOn(date string) ([]Task, error)

	PushSubs() ([]PushSub, error)
	PutPushSub(PushSub) error
	DeletePushSub(endpoint string) error
	TouchPushSub(endpoint string, lastOK int64) error
	// LogReminder records that slot fired on localDate. It reports false when
	// a row already exists, which is what stops a second send.
	LogReminder(slot int, localDate string, sentAt int64) (bool, error)

	ReviewDone(weekStart string) (bool, error)
	LogReview(weekStart string, doneAt int64) error

	Session(tokenHash string) (Session, error)
	InsertSession(Session) error
	TouchSession(tokenHash string, lastSeen int64) error
	Sessions() ([]Session, error)
	SetCalendarToken(token string) error
	DeleteSession(tokenHash string) error
}
