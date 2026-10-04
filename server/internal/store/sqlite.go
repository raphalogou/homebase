package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
)

// SQLite is the Store backed by database/sql. Tests use it with an in-memory
// database.
type SQLite struct {
	db *sql.DB
}

// NewSQLite wraps an open, migrated database.
func NewSQLite(db *sql.DB) *SQLite {
	return &SQLite{db: db}
}

func (s *SQLite) Close() error { return s.db.Close() }

func (s *SQLite) Tx(ctx context.Context, fn func(Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := fn(&sqliteTx{ctx: ctx, tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

type sqliteTx struct {
	ctx context.Context
	tx  *sql.Tx
	sp  int
}

func (t *sqliteTx) exec(query string, args ...any) error {
	_, err := t.tx.ExecContext(t.ctx, query, args...)
	return err
}

func (t *sqliteTx) row(query string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(t.ctx, query, args...)
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (t *sqliteTx) NextRev() (int64, error) {
	var rev int64
	err := t.row(`UPDATE meta SET rev = rev + 1 WHERE id = 1 RETURNING rev`).Scan(&rev)
	return rev, err
}

func (t *sqliteTx) Rev() (int64, error) {
	var rev int64
	err := t.row(`SELECT rev FROM meta WHERE id = 1`).Scan(&rev)
	return rev, err
}

func (t *sqliteTx) Savepoint(fn func() error) error {
	t.sp++
	name := fmt.Sprintf("op%d", t.sp)
	if err := t.exec("SAVEPOINT " + name); err != nil {
		return err
	}
	if err := fn(); err != nil {
		if rbErr := t.exec("ROLLBACK TO " + name); rbErr != nil {
			return errors.Join(err, rbErr)
		}
		if relErr := t.exec("RELEASE " + name); relErr != nil {
			return errors.Join(err, relErr)
		}
		return err
	}
	return t.exec("RELEASE " + name)
}

// Settings

func (t *sqliteTx) Settings() (Settings, error) {
	var s Settings
	err := t.row(`SELECT tz, week_start, calendar_token, last_rollover FROM settings WHERE id = 1`).
		Scan(&s.TZ, &s.WeekStart, &s.CalendarToken, &s.LastRollover)
	return s, notFound(err)
}

func (t *sqliteTx) InsertSettings(s Settings) error {
	return t.exec(`INSERT INTO settings(id, tz, week_start, calendar_token, last_rollover) VALUES (1, ?, ?, ?, ?)`,
		s.TZ, s.WeekStart, s.CalendarToken, s.LastRollover)
}

func (t *sqliteTx) SetLastRollover(date string) error {
	return t.exec(`UPDATE settings SET last_rollover = ? WHERE id = 1`, date)
}

// Reminders

const reminderCols = `slot, enabled, at_local, kind, updated_at, rev`

func scanReminder(sc interface{ Scan(...any) error }) (Reminder, error) {
	var r Reminder
	err := sc.Scan(&r.Slot, &r.Enabled, &r.AtLocal, &r.Kind, &r.UpdatedAt, &r.Rev)
	return r, err
}

func (t *sqliteTx) Reminders() ([]Reminder, error) {
	return queryAll(t, `SELECT `+reminderCols+` FROM reminders ORDER BY slot`, scanReminder)
}

func (t *sqliteTx) PutReminder(r Reminder) error {
	return t.exec(`INSERT INTO reminders(`+reminderCols+`) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(slot) DO UPDATE SET enabled = excluded.enabled, at_local = excluded.at_local,
		kind = excluded.kind, updated_at = excluded.updated_at, rev = excluded.rev`,
		r.Slot, r.Enabled, r.AtLocal, r.Kind, r.UpdatedAt, r.Rev)
}

// Goals

const goalCols = `id, title, notes, status, target_date, sort_key, created_at, updated_at, deleted_at, rev`

func scanGoal(sc interface{ Scan(...any) error }) (Goal, error) {
	var g Goal
	err := sc.Scan(&g.ID, &g.Title, &g.Notes, &g.Status, &g.TargetDate, &g.SortKey,
		&g.CreatedAt, &g.UpdatedAt, &g.DeletedAt, &g.Rev)
	return g, err
}

func (t *sqliteTx) Goal(id string) (Goal, error) {
	g, err := scanGoal(t.row(`SELECT `+goalCols+` FROM goals WHERE id = ?`, id))
	return g, notFound(err)
}

func (t *sqliteTx) PutGoal(g Goal) error {
	return t.exec(`INSERT INTO goals(`+goalCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET title = excluded.title, notes = excluded.notes,
		status = excluded.status, target_date = excluded.target_date, sort_key = excluded.sort_key,
		created_at = excluded.created_at, updated_at = excluded.updated_at,
		deleted_at = excluded.deleted_at, rev = excluded.rev`,
		g.ID, g.Title, g.Notes, g.Status, g.TargetDate, g.SortKey, g.CreatedAt, g.UpdatedAt, g.DeletedAt, g.Rev)
}

// Projects

const projectCols = `id, goal_id, title, notes, status, due, created_at, updated_at, deleted_at, rev`

func scanProject(sc interface{ Scan(...any) error }) (Project, error) {
	var p Project
	err := sc.Scan(&p.ID, &p.GoalID, &p.Title, &p.Notes, &p.Status, &p.Due,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt, &p.Rev)
	return p, err
}

func (t *sqliteTx) Project(id string) (Project, error) {
	p, err := scanProject(t.row(`SELECT `+projectCols+` FROM projects WHERE id = ?`, id))
	return p, notFound(err)
}

func (t *sqliteTx) PutProject(p Project) error {
	return t.exec(`INSERT INTO projects(`+projectCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET goal_id = excluded.goal_id, title = excluded.title,
		notes = excluded.notes, status = excluded.status, due = excluded.due,
		created_at = excluded.created_at, updated_at = excluded.updated_at,
		deleted_at = excluded.deleted_at, rev = excluded.rev`,
		p.ID, p.GoalID, p.Title, p.Notes, p.Status, p.Due, p.CreatedAt, p.UpdatedAt, p.DeletedAt, p.Rev)
}

func (t *sqliteTx) ProjectsOfGoal(goalID string) ([]Project, error) {
	return queryAll(t, `SELECT `+projectCols+` FROM projects WHERE goal_id = ? AND deleted_at IS NULL ORDER BY id`,
		scanProject, goalID)
}

// Repeats

const repeatCols = `id, freq, every, weekdays, mode, until, created_at, updated_at, deleted_at, rev`

func scanRepeat(sc interface{ Scan(...any) error }) (Repeat, error) {
	var r Repeat
	err := sc.Scan(&r.ID, &r.Freq, &r.Every, &r.Weekdays, &r.Mode, &r.Until,
		&r.CreatedAt, &r.UpdatedAt, &r.DeletedAt, &r.Rev)
	return r, err
}

func (t *sqliteTx) Repeat(id string) (Repeat, error) {
	r, err := scanRepeat(t.row(`SELECT `+repeatCols+` FROM repeats WHERE id = ?`, id))
	return r, notFound(err)
}

func (t *sqliteTx) PutRepeat(r Repeat) error {
	return t.exec(`INSERT INTO repeats(`+repeatCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET freq = excluded.freq, every = excluded.every,
		weekdays = excluded.weekdays, mode = excluded.mode, until = excluded.until,
		created_at = excluded.created_at, updated_at = excluded.updated_at,
		deleted_at = excluded.deleted_at, rev = excluded.rev`,
		r.ID, r.Freq, r.Every, r.Weekdays, r.Mode, r.Until, r.CreatedAt, r.UpdatedAt, r.DeletedAt, r.Rev)
}

// Tasks

const taskCols = `id, project_id, goal_id, title, notes, status, due, planned_on, plan_rank, slipped,
	repeat_id, done_at, created_at, updated_at, deleted_at, rev`

func scanTask(sc interface{ Scan(...any) error }) (Task, error) {
	var t Task
	err := sc.Scan(&t.ID, &t.ProjectID, &t.GoalID, &t.Title, &t.Notes, &t.Status, &t.Due, &t.PlannedOn,
		&t.PlanRank, &t.Slipped, &t.RepeatID, &t.DoneAt, &t.CreatedAt, &t.UpdatedAt, &t.DeletedAt, &t.Rev)
	return t, err
}

func (t *sqliteTx) Task(id string) (Task, error) {
	task, err := scanTask(t.row(`SELECT `+taskCols+` FROM tasks WHERE id = ?`, id))
	return task, notFound(err)
}

func (t *sqliteTx) PutTask(k Task) error {
	return t.exec(`INSERT INTO tasks(`+taskCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET project_id = excluded.project_id, goal_id = excluded.goal_id,
		title = excluded.title, notes = excluded.notes, status = excluded.status, due = excluded.due,
		planned_on = excluded.planned_on, plan_rank = excluded.plan_rank, slipped = excluded.slipped,
		repeat_id = excluded.repeat_id, done_at = excluded.done_at, created_at = excluded.created_at,
		updated_at = excluded.updated_at, deleted_at = excluded.deleted_at, rev = excluded.rev`,
		k.ID, k.ProjectID, k.GoalID, k.Title, k.Notes, k.Status, k.Due, k.PlannedOn, k.PlanRank, k.Slipped,
		k.RepeatID, k.DoneAt, k.CreatedAt, k.UpdatedAt, k.DeletedAt, k.Rev)
}

func (t *sqliteTx) CountPlanned(date, exceptID string) (int, error) {
	var n int
	err := t.row(`SELECT count(*) FROM tasks
		WHERE planned_on = ? AND id != ? AND deleted_at IS NULL AND status != 'dropped'`, date, exceptID).Scan(&n)
	return n, err
}

func (t *sqliteTx) OpenTasksPlannedBefore(date string) ([]Task, error) {
	return queryAll(t, `SELECT `+taskCols+` FROM tasks
		WHERE planned_on < ? AND status = 'open' AND deleted_at IS NULL ORDER BY id`, scanTask, date)
}

func (t *sqliteTx) HasOpenInstance(repeatID, exceptID string) (bool, error) {
	var n int
	err := t.row(`SELECT count(*) FROM tasks
		WHERE repeat_id = ? AND id != ? AND status = 'open' AND deleted_at IS NULL`, repeatID, exceptID).Scan(&n)
	return n > 0, err
}

func (t *sqliteTx) TasksOfGoal(goalID string) ([]Task, error) {
	return queryAll(t, `SELECT `+taskCols+` FROM tasks WHERE goal_id = ? AND deleted_at IS NULL ORDER BY id`,
		scanTask, goalID)
}

func (t *sqliteTx) TasksOfProject(projectID string) ([]Task, error) {
	return queryAll(t, `SELECT `+taskCols+` FROM tasks WHERE project_id = ? AND deleted_at IS NULL ORDER BY id`,
		scanTask, projectID)
}

// Attachments

const attachmentCols = `id, goal_id, project_id, task_id, kind, name, url, body, file_sha,
	created_at, updated_at, deleted_at, rev`

func scanAttachment(sc interface{ Scan(...any) error }) (Attachment, error) {
	var a Attachment
	err := sc.Scan(&a.ID, &a.GoalID, &a.ProjectID, &a.TaskID, &a.Kind, &a.Name, &a.URL, &a.Body, &a.FileSHA,
		&a.CreatedAt, &a.UpdatedAt, &a.DeletedAt, &a.Rev)
	return a, err
}

func (t *sqliteTx) Attachment(id string) (Attachment, error) {
	a, err := scanAttachment(t.row(`SELECT `+attachmentCols+` FROM attachments WHERE id = ?`, id))
	return a, notFound(err)
}

func (t *sqliteTx) PutAttachment(a Attachment) error {
	return t.exec(`INSERT INTO attachments(`+attachmentCols+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET goal_id = excluded.goal_id, project_id = excluded.project_id,
		task_id = excluded.task_id, kind = excluded.kind, name = excluded.name, url = excluded.url,
		body = excluded.body, file_sha = excluded.file_sha, created_at = excluded.created_at,
		updated_at = excluded.updated_at, deleted_at = excluded.deleted_at, rev = excluded.rev`,
		a.ID, a.GoalID, a.ProjectID, a.TaskID, a.Kind, a.Name, a.URL, a.Body, a.FileSHA,
		a.CreatedAt, a.UpdatedAt, a.DeletedAt, a.Rev)
}

func (t *sqliteTx) AttachmentsOf(kind OwnerKind, ownerID string) ([]Attachment, error) {
	var col string
	switch kind {
	case OwnerGoal:
		col = "goal_id"
	case OwnerProject:
		col = "project_id"
	case OwnerTask:
		col = "task_id"
	default:
		return nil, fmt.Errorf("unknown owner kind %q", kind)
	}
	return queryAll(t, `SELECT `+attachmentCols+` FROM attachments WHERE `+col+` = ? AND deleted_at IS NULL ORDER BY id`,
		scanAttachment, ownerID)
}

func (t *sqliteTx) FileExists(sha string) (bool, error) {
	var n int
	err := t.row(`SELECT count(*) FROM files WHERE sha = ?`, sha).Scan(&n)
	return n > 0, err
}

func (t *sqliteTx) InsertFile(f File) error {
	return t.exec(`INSERT INTO files(sha, mime, size, created_at) VALUES (?, ?, ?, ?) ON CONFLICT(sha) DO NOTHING`,
		f.SHA, f.Mime, f.Size, f.CreatedAt)
}

func (t *sqliteTx) File(sha string) (File, error) {
	var f File
	err := t.row(`SELECT sha, mime, size, created_at FROM files WHERE sha = ?`, sha).
		Scan(&f.SHA, &f.Mime, &f.Size, &f.CreatedAt)
	return f, notFound(err)
}

func (t *sqliteTx) OrphanFiles(cutoff int64) ([]string, error) {
	return queryAll(t, `SELECT sha FROM files f WHERE f.created_at < ?
		AND NOT EXISTS (SELECT 1 FROM attachments a WHERE a.file_sha = f.sha
			AND (a.deleted_at IS NULL OR a.updated_at >= ?))
		ORDER BY sha`,
		func(sc interface{ Scan(...any) error }) (string, error) {
			var s string
			err := sc.Scan(&s)
			return s, err
		}, cutoff, cutoff)
}

// Changes

func (t *sqliteTx) ChangesSince(since int64, limit int) (Changes, bool, int64, error) {
	c := EmptyChanges()
	// Fetch one row past the limit from each table, then keep the lowest
	// revisions overall. Revisions are unique, so the cut is exact.
	lim := -1
	if limit > 0 {
		lim = limit + 1
	}
	var err error
	if c.Goals, err = queryAll(t, `SELECT `+goalCols+` FROM goals WHERE rev > ? ORDER BY rev LIMIT ?`, scanGoal, since, lim); err != nil {
		return c, false, 0, err
	}
	if c.Projects, err = queryAll(t, `SELECT `+projectCols+` FROM projects WHERE rev > ? ORDER BY rev LIMIT ?`, scanProject, since, lim); err != nil {
		return c, false, 0, err
	}
	if c.Tasks, err = queryAll(t, `SELECT `+taskCols+` FROM tasks WHERE rev > ? ORDER BY rev LIMIT ?`, scanTask, since, lim); err != nil {
		return c, false, 0, err
	}
	if c.Repeats, err = queryAll(t, `SELECT `+repeatCols+` FROM repeats WHERE rev > ? ORDER BY rev LIMIT ?`, scanRepeat, since, lim); err != nil {
		return c, false, 0, err
	}
	if c.Attachments, err = queryAll(t, `SELECT `+attachmentCols+` FROM attachments WHERE rev > ? ORDER BY rev LIMIT ?`, scanAttachment, since, lim); err != nil {
		return c, false, 0, err
	}
	if c.Reminders, err = queryAll(t, `SELECT `+reminderCols+` FROM reminders WHERE rev > ? ORDER BY rev LIMIT ?`, scanReminder, since, lim); err != nil {
		return c, false, 0, err
	}

	revs := allRevs(c)
	sort.Slice(revs, func(i, j int) bool { return revs[i] < revs[j] })
	if limit <= 0 || len(revs) <= limit {
		last := since
		if len(revs) > 0 {
			last = revs[len(revs)-1]
		}
		return c, false, last, nil
	}

	cut := revs[limit-1]
	keep := func(rev int64) bool { return rev <= cut }
	c.Goals = filter(c.Goals, func(r Goal) bool { return keep(r.Rev) })
	c.Projects = filter(c.Projects, func(r Project) bool { return keep(r.Rev) })
	c.Tasks = filter(c.Tasks, func(r Task) bool { return keep(r.Rev) })
	c.Repeats = filter(c.Repeats, func(r Repeat) bool { return keep(r.Rev) })
	c.Attachments = filter(c.Attachments, func(r Attachment) bool { return keep(r.Rev) })
	c.Reminders = filter(c.Reminders, func(r Reminder) bool { return keep(r.Rev) })
	return c, true, cut, nil
}

func allRevs(c Changes) []int64 {
	var revs []int64
	for _, r := range c.Goals {
		revs = append(revs, r.Rev)
	}
	for _, r := range c.Projects {
		revs = append(revs, r.Rev)
	}
	for _, r := range c.Tasks {
		revs = append(revs, r.Rev)
	}
	for _, r := range c.Repeats {
		revs = append(revs, r.Rev)
	}
	for _, r := range c.Attachments {
		revs = append(revs, r.Rev)
	}
	for _, r := range c.Reminders {
		revs = append(revs, r.Rev)
	}
	return revs
}

func (t *sqliteTx) UpdateSettings(tz string, weekStart int) error {
	return t.exec(`UPDATE settings SET tz = ?, week_start = ? WHERE id = 1`, tz, weekStart)
}

func (t *sqliteTx) OpenGoals() ([]Goal, error) {
	return queryAll(t, `SELECT `+goalCols+` FROM goals WHERE status = 'open' AND deleted_at IS NULL
		ORDER BY sort_key, created_at, id`, scanGoal)
}

func (t *sqliteTx) TasksPlannedOn(date string) ([]Task, error) {
	return queryAll(t, `SELECT `+taskCols+` FROM tasks WHERE planned_on = ? AND deleted_at IS NULL
		AND status != 'dropped' ORDER BY plan_rank IS NULL, plan_rank, created_at, id`, scanTask, date)
}

// Push subscriptions and the reminder log

func (t *sqliteTx) PushSubs() ([]PushSub, error) {
	return queryAll(t, `SELECT endpoint, p256dh, auth, label, created_at, last_ok FROM push_subs ORDER BY created_at`,
		func(sc interface{ Scan(...any) error }) (PushSub, error) {
			var p PushSub
			err := sc.Scan(&p.Endpoint, &p.P256DH, &p.Auth, &p.Label, &p.CreatedAt, &p.LastOK)
			return p, err
		})
}

func (t *sqliteTx) PutPushSub(p PushSub) error {
	return t.exec(`INSERT INTO push_subs(endpoint, p256dh, auth, label, created_at, last_ok) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(endpoint) DO UPDATE SET p256dh = excluded.p256dh, auth = excluded.auth, label = excluded.label`,
		p.Endpoint, p.P256DH, p.Auth, p.Label, p.CreatedAt, p.LastOK)
}

func (t *sqliteTx) DeletePushSub(endpoint string) error {
	return t.exec(`DELETE FROM push_subs WHERE endpoint = ?`, endpoint)
}

func (t *sqliteTx) TouchPushSub(endpoint string, lastOK int64) error {
	return t.exec(`UPDATE push_subs SET last_ok = ? WHERE endpoint = ?`, lastOK, endpoint)
}

func (t *sqliteTx) LogReminder(slot int, localDate string, sentAt int64) (bool, error) {
	res, err := t.tx.ExecContext(t.ctx, `INSERT INTO reminder_log(slot, local_date, sent_at) VALUES (?, ?, ?)
		ON CONFLICT(slot, local_date) DO NOTHING`, slot, localDate, sentAt)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (t *sqliteTx) ReviewDone(weekStart string) (bool, error) {
	var n int
	err := t.row(`SELECT count(*) FROM review_log WHERE week_start = ?`, weekStart).Scan(&n)
	return n > 0, err
}

func (t *sqliteTx) LogReview(weekStart string, doneAt int64) error {
	return t.exec(`INSERT INTO review_log(week_start, done_at) VALUES (?, ?)
		ON CONFLICT(week_start) DO UPDATE SET done_at = excluded.done_at`, weekStart, doneAt)
}

// Account

func (t *sqliteTx) Account() (Account, error) {
	var a Account
	err := t.row(`SELECT username, passphrase_hash, changed_at FROM account WHERE id = 1`).
		Scan(&a.Username, &a.PassphraseHash, &a.ChangedAt)
	return a, notFound(err)
}

func (t *sqliteTx) InsertAccount(a Account) error {
	return t.exec(`INSERT INTO account(id, username, passphrase_hash, changed_at) VALUES (1, ?, ?, ?)`,
		a.Username, a.PassphraseHash, a.ChangedAt)
}

func (t *sqliteTx) SetUsername(username string) error {
	return t.exec(`UPDATE account SET username = ? WHERE id = 1`, username)
}

func (t *sqliteTx) SetPassphraseHash(hash string, changedAt int64) error {
	return t.exec(`UPDATE account SET passphrase_hash = ?, changed_at = ? WHERE id = 1`, hash, changedAt)
}

// Sessions

func (t *sqliteTx) Session(tokenHash string) (Session, error) {
	var s Session
	err := t.row(`SELECT token_hash, label, created_at, last_seen FROM sessions WHERE token_hash = ?`, tokenHash).
		Scan(&s.TokenHash, &s.Label, &s.CreatedAt, &s.LastSeen)
	return s, notFound(err)
}

func (t *sqliteTx) InsertSession(s Session) error {
	return t.exec(`INSERT INTO sessions(token_hash, label, created_at, last_seen) VALUES (?, ?, ?, ?)`,
		s.TokenHash, s.Label, s.CreatedAt, s.LastSeen)
}

func (t *sqliteTx) TouchSession(tokenHash string, lastSeen int64) error {
	return t.exec(`UPDATE sessions SET last_seen = ? WHERE token_hash = ?`, lastSeen, tokenHash)
}

func (t *sqliteTx) Sessions() ([]Session, error) {
	return queryAll(t, `SELECT token_hash, label, created_at, last_seen FROM sessions ORDER BY last_seen DESC`,
		func(sc interface{ Scan(...any) error }) (Session, error) {
			var s Session
			err := sc.Scan(&s.TokenHash, &s.Label, &s.CreatedAt, &s.LastSeen)
			return s, err
		})
}

func (t *sqliteTx) SetCalendarToken(token string) error {
	return t.exec(`UPDATE settings SET calendar_token = ? WHERE id = 1`, token)
}

func (t *sqliteTx) DeleteSession(tokenHash string) error {
	return t.exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
}

func (t *sqliteTx) DeleteSessionsExcept(keepHash string) error {
	return t.exec(`DELETE FROM sessions WHERE token_hash <> ?`, keepHash)
}

// Helpers

func queryAll[T any](t *sqliteTx, query string, scan func(interface{ Scan(...any) error }) (T, error), args ...any) ([]T, error) {
	rows, err := t.tx.QueryContext(t.ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func filter[T any](in []T, keep func(T) bool) []T {
	out := in[:0]
	for _, v := range in {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}
