# Homebase specification

Source of truth for the data model, sync protocol and API. If code and this file disagree, fix one of them in the same change.

## 1. Conventions

- **IDs:** client-generated ULIDs (26 chars). Items created offline never collide.
- **Local-day dates:** `due`, `planned_on`, `target_date`, `until` are `YYYY-MM-DD` text with no time zone. They mean that calendar day wherever the user is.
- **Timestamps:** UTC milliseconds, integers. The user's zone lives in `settings.tz` and is used only for rollover, reminders and the calendar feed.
- **Revision:** one counter in `meta.rev`. Every write to a synced table increments it in the same transaction and stamps the row's `rev`.
- **Deletion:** soft. `deleted_at` is set, the row stays as a tombstone for 90 days.
- **Status:** goals and projects: `open | paused | done | dropped`. Tasks: `open | done | dropped`.
- **Ordering:** `plan_rank` is a float. Dragging sets the rank to the midpoint of the new neighbours; renumber the day when the gap falls below 1e-6.
- **Limits:** titles 300 chars, notes and note attachments 20,000, files 25 MB.
- **Errors:** `{"error":{"code","message"}}` with a matching HTTP status. Codes: `invalid` (400), `unauthorized` (401), `not_found` (404), `too_large` (413), `rate_limited` (429).

## 2. Schema (`server/migrations/0001_init.sql`)

Set `PRAGMA foreign_keys=ON` and `journal_mode=WAL` when each connection opens, not inside the migration. The application, not triggers, stamps `rev` and `updated_at`.

```sql
CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL);

CREATE TABLE meta (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  rev INTEGER NOT NULL DEFAULT 0
);
INSERT INTO meta(id, rev) VALUES (1, 0);

CREATE TABLE settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  tz TEXT NOT NULL DEFAULT 'UTC',
  week_start INTEGER NOT NULL DEFAULT 1 CHECK (week_start IN (0, 1)),  -- 1 Monday, 0 Sunday
  calendar_token TEXT NOT NULL,
  last_rollover TEXT                                                   -- local date of last rollover
);

CREATE TABLE goals (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 300),
  notes TEXT NOT NULL DEFAULT '' CHECK (length(notes) <= 20000),
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','paused','done','dropped')),
  target_date TEXT CHECK (target_date IS NULL OR target_date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  sort_key REAL NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL,
  deleted_at INTEGER, rev INTEGER NOT NULL
);

CREATE TABLE projects (
  id TEXT PRIMARY KEY,
  goal_id TEXT REFERENCES goals(id),                -- NULL = free-standing project
  title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 300),
  notes TEXT NOT NULL DEFAULT '' CHECK (length(notes) <= 20000),
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','paused','done','dropped')),
  due TEXT CHECK (due IS NULL OR due GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL,
  deleted_at INTEGER, rev INTEGER NOT NULL
);

CREATE TABLE repeats (
  id TEXT PRIMARY KEY,
  freq TEXT NOT NULL CHECK (freq IN ('day','week','month')),
  every INTEGER NOT NULL DEFAULT 1 CHECK (every >= 1),
  weekdays INTEGER CHECK (weekdays IS NULL OR weekdays BETWEEN 1 AND 127),  -- Mon=1 ... Sun=64
  mode TEXT NOT NULL DEFAULT 'after_done' CHECK (mode IN ('fixed','after_done')),
  until TEXT CHECK (until IS NULL OR until GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL,
  deleted_at INTEGER, rev INTEGER NOT NULL
);

CREATE TABLE tasks (
  id TEXT PRIMARY KEY,
  project_id TEXT REFERENCES projects(id),
  goal_id TEXT REFERENCES goals(id),                -- only for a task directly under a goal
  title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 300),
  notes TEXT NOT NULL DEFAULT '' CHECK (length(notes) <= 20000),
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','done','dropped')),
  due TEXT CHECK (due IS NULL OR due GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  planned_on TEXT CHECK (planned_on IS NULL OR planned_on GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
  plan_rank REAL,
  slipped INTEGER NOT NULL DEFAULT 0 CHECK (slipped >= 0),   -- server-owned
  repeat_id TEXT REFERENCES repeats(id),
  done_at INTEGER,
  created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL,
  deleted_at INTEGER, rev INTEGER NOT NULL,
  CHECK (project_id IS NULL OR goal_id IS NULL),
  CHECK ((status = 'done') = (done_at IS NOT NULL))
);

CREATE TABLE files (
  sha TEXT PRIMARY KEY,                              -- sha256 hex, also the on-disk name
  mime TEXT NOT NULL, size INTEGER NOT NULL CHECK (size <= 26214400),
  created_at INTEGER NOT NULL
);

CREATE TABLE attachments (
  id TEXT PRIMARY KEY,
  goal_id TEXT REFERENCES goals(id),
  project_id TEXT REFERENCES projects(id),
  task_id TEXT REFERENCES tasks(id),
  kind TEXT NOT NULL CHECK (kind IN ('link','note','file')),
  name TEXT NOT NULL DEFAULT '' CHECK (length(name) <= 300),
  url TEXT CHECK (url IS NULL OR url GLOB 'http*://*' OR url GLOB 'mailto:*'),
  body TEXT CHECK (body IS NULL OR length(body) <= 20000),
  file_sha TEXT REFERENCES files(sha),
  created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL,
  deleted_at INTEGER, rev INTEGER NOT NULL,
  CHECK ((goal_id IS NOT NULL) + (project_id IS NOT NULL) + (task_id IS NOT NULL) = 1),
  CHECK ((kind = 'link' AND url IS NOT NULL) OR (kind = 'note' AND body IS NOT NULL)
         OR (kind = 'file' AND file_sha IS NOT NULL))
);

CREATE TABLE reminders (
  slot INTEGER PRIMARY KEY CHECK (slot BETWEEN 1 AND 3),
  enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0,1)),
  at_local TEXT NOT NULL CHECK (at_local GLOB '[0-2][0-9]:[0-5][0-9]'),
  kind TEXT NOT NULL CHECK (kind IN ('focus','checkin','wrap')),
  updated_at INTEGER NOT NULL, rev INTEGER NOT NULL
);
INSERT INTO reminders(slot, enabled, at_local, kind, updated_at, rev) VALUES
  (1, 1, '08:00', 'focus', 0, 0), (2, 0, '13:00', 'checkin', 0, 0), (3, 1, '20:00', 'wrap', 0, 0);

CREATE TABLE reminder_log (
  slot INTEGER NOT NULL, local_date TEXT NOT NULL, sent_at INTEGER NOT NULL,
  PRIMARY KEY (slot, local_date)                     -- guarantees one send per slot per day
);

CREATE TABLE push_subs (
  endpoint TEXT PRIMARY KEY, p256dh TEXT NOT NULL, auth TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL, last_ok INTEGER
);

CREATE TABLE sessions (
  token_hash TEXT PRIMARY KEY,                       -- sha256 of the cookie value
  label TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL, last_seen INTEGER NOT NULL
);

CREATE TABLE review_log (week_start TEXT PRIMARY KEY, done_at INTEGER NOT NULL);

CREATE INDEX tasks_planned  ON tasks(planned_on) WHERE deleted_at IS NULL;
CREATE INDEX tasks_due      ON tasks(due)        WHERE deleted_at IS NULL AND status = 'open';
CREATE INDEX tasks_project  ON tasks(project_id);
CREATE INDEX tasks_goal     ON tasks(goal_id);
CREATE INDEX tasks_rev      ON tasks(rev);
CREATE INDEX projects_goal  ON projects(goal_id);
CREATE INDEX projects_rev   ON projects(rev);
CREATE INDEX goals_rev      ON goals(rev);
CREATE INDEX attach_goal    ON attachments(goal_id);
CREATE INDEX attach_project ON attachments(project_id);
CREATE INDEX attach_task    ON attachments(task_id);
CREATE INDEX attach_rev     ON attachments(rev);
CREATE INDEX repeats_rev    ON repeats(rev);
```

`settings` gets its single row at first run, because the calendar token must be random. Migrations are append-only: never edit `0001`, add `0002`.

## 3. Sync protocol

Revision-based: every change gets a number, and a device asks for everything newer than the last number it saw. Almost every write is a plain row upsert through `/api/sync`. Only multi-row operations get their own endpoint.

1. **Pull.** `GET /api/sync?since=N&limit=500` returns every row with `rev > N` from goals, projects, tasks, repeats, attachments and reminders, tombstones included, plus the current `rev`. `more:true` means ask again from the new `rev`.
2. **Push.** `POST /api/sync` takes `{base, ops}`. Each op is `{op:"upsert"|"delete", table, id, row, updatedAt, cascade?}`. The server applies all ops in one transaction, in order, and answers `{rev, applied, rejected, changes}`. `changes` holds rows newer than `base`, so the client needs no second pull.
3. **Offline queue.** Changes apply to the local copy instantly and go to an IndexedDB outbox. Ops are coalesced per row (latest upsert wins), retried with backoff, and idempotent (upsert by ID).
4. **When it runs.** App start, focus, browser `online`, 800 ms after a local change, every 60 s while visible, and when a push message carries the `sync` hint.
5. **Server-owned fields.** The server stamps `rev`, `slipped` and `done_at` itself. It ignores client values for the first two and fills `done_at` when a task is completed without one.
6. **Recurrence runs in the apply step.** When an op moves a task with a `repeat_id` from `open` to `done` or `dropped`, the server creates the next instance in the same transaction, so completing a repeating chore offline still works.

| Situation | Result |
| --- | --- |
| Same row edited on two devices | Row-level last write wins by `updatedAt`. No field merge. |
| Delete on one device, edit on another | The later `updatedAt` wins. A later edit clears `deleted_at`. |
| Delete a project | `cascade:"delete"` tombstones its tasks and attachments. `cascade:"detach"` clears `project_id` so tasks become standalone. The UI asks each time. |
| Delete a goal | `cascade:"delete"` removes its projects and tasks. `cascade:"detach"` frees its projects and makes its direct tasks standalone. |
| Client clock far ahead | An `updatedAt` more than 5 minutes in the future is clamped to server time. |
| Plan a fourth task for one day | Rejected with `invalid`, reason `day_full`. |

### Details

- **Row shape.** Rows travel as JSON with camelCase names for the columns (`goalId`, `plannedOn`, `planRank`, `doneAt`, `updatedAt`, `deletedAt`, `fileSha`, `atLocal`, ...). Nullable columns are always present, as `null` when empty. The same shape is used in pulls, in op rows and in `changes`.
- **Revisions.** Every row write takes its own `meta.rev` value, so revisions are unique. A pull page holds the `limit` rows with the lowest revisions; with `more:true`, `rev` is the revision of the last row in the page, otherwise it is `meta.rev`. First-run setup gives each seeded reminder a revision so that a first pull returns them.
- **Op rows.** An upsert replaces the whole row. `id` in the row may be left out but must match the op's `id` if present. Server-owned fields (`rev`, `slipped`, `createdAt` of an existing row, `deletedAt`) are ignored. `doneAt` is kept if the client sends a sensible one, filled with the op's time otherwise, and cleared when the status is not `done`.
- **Ties.** An op whose `updatedAt` equals the stored row's is treated as already applied (a re-push), not as a conflict.
- **Rejections.** `rejected[].reason` is `invalid` (validation or a missing parent row), `stale` (the server has a newer version), or `day_full`. For every rejected op, `changes` also includes the server's current copy of that row, if it exists, even when its `rev` is not above `base`, so the client can undo its optimistic change.
- **Deletes.** Deleting a row the server never saw, or one already deleted, is applied as a no-op. Deleting a task also tombstones its attachments. A goal or project delete without `cascade` detaches. Rows touched by a cascade keep the later of their own `updatedAt` and the delete's.
- **Limits.** At most 1000 ops per push (`too_large` otherwise). Pull `limit` defaults to 500, maximum 1000.

## 4. API

JSON over HTTPS with a session cookie. Every state-changing request needs `Content-Type: application/json` (multipart for upload) and the header `X-Homebase: 1`.

| Endpoint | Request | Response |
| --- | --- | --- |
| `POST /api/login` | `{passphrase}` | 204 and cookie `hb_session` (HttpOnly, Secure, SameSite=Lax, 180 days, sliding). Five failures per 10 min per address, then 429. |
| `POST /api/logout` | none | 204 |
| `GET /api/me` | none | `{tz, weekStart, rev}` |
| `GET /api/sync?since=N&limit=500` | none | `{rev, more, goals[], projects[], tasks[], repeats[], attachments[], reminders[]}` |
| `POST /api/sync` | `{base, ops[]}` | `{rev, applied[], rejected[{id,reason}], changes}` |
| `POST /api/promote` | `{taskId, goalId?}` | `{project, removedTaskId}`. One transaction: new project from an Inbox task, notes and attachments moved, task tombstoned. The project takes the task's title, notes and `due`; without `goalId` it keeps the task's goal. A task already in a project is `invalid`. |
| `GET /api/review` | none | `{weekStart, doneCount, doneByGoal[], quiet[{kind,id,title,lastActivity}], goalsWithNothing[]}` |
| `POST /api/review/complete` | `{weekStart, decisions[{kind,id,action}]}`, action is `keep`, `pause` or `drop` | `{rev}`. Applies decisions, writes `review_log`. |
| `POST /api/files` | multipart: `file`, `ownerKind`, `ownerId`, optional `name` | The new `attachments` row. Stored by SHA-256. |
| `GET /api/files/{sha}` | none | The file. Images and PDFs inline, others as download. Always `X-Content-Type-Options: nosniff`. |
| `GET /api/settings`, `PUT /api/settings` | `{tz, weekStart}` | The settings. `tz` must be a valid IANA name. |
| `PUT /api/reminders` | `[{slot, enabled, atLocal, kind}]`, at most 3 | The saved rows |
| `GET /api/push/key` | none | `{publicKey}` (VAPID, base64url) |
| `POST /api/push/subscribe` | `{endpoint, keys:{p256dh, auth}, label}` | 204 |
| `POST /api/push/unsubscribe` | `{endpoint}` | 204 |
| `POST /api/push/test` | none | `{sent, failed}` |
| `GET /calendar/{token}.ics` | none, no cookie | iCalendar feed. 404 for a wrong token. |
| `POST /api/calendar/rotate` | none | `{url}`. The old link stops at once. |
| `GET /healthz` | none | `ok`, no auth |

Links and notes are ordinary attachment rows written through `/api/sync`. Removing any attachment is a `delete` op.

## 5. Behaviour rules

All of these live on the server so every device sees the same result.

- **Today's three.** At most 3 tasks per `planned_on` date, counting done ones. Applies to future days too. Dropped and deleted tasks do not count.
- **Inbox.** An open, live task with no project, no goal, no `due` and no `planned_on`.
- **Daily rollover.** At 00:05 in `settings.tz`, and on the first sync of a new day if missed. Every open task with `planned_on` before today gets `planned_on` and `plan_rank` cleared and `slipped` increased by 1. Nothing turns red. The rollover bumps `rev` but leaves `updated_at` alone, so an edit made offline before midnight (ticking the task done at 23:59) still wins when it arrives. An open task pushed with a `planned_on` before today has its plan cleared on arrival; `slipped` is not increased again.
- **Reconsider pile.** Open tasks with `slipped >= 2`, shown as "Moved a few times". Completing or dropping a task ends the count.
- **Overdue.** A past `due` stays visible as text such as "2d overdue". Only `planned_on` rolls over, never `due`.
- **Goal progress.** Done tasks divided by all non-dropped tasks, counted from the goal's projects and from tasks directly under it.

### Recurrence

- A repeating task is one row with a `repeat_id`. Completing or dropping it creates the successor. Dropping counts as skipping this time.
- `after_done`: next `due` is the completion date plus `every` days, weeks or months. Right for chores.
- `fixed`: next `due` is the next date after the old `due` matching the rule (weekday mask or day of month). Without an old `due`, the completion day is used. With a weekday mask and `every` above 1, weeks run Monday to Sunday and only every N-th week from the old `due`'s week counts.
- **Month ends.** Adding months clamps to the end of a shorter month (31 January plus one month is 28 or 29 February). For `fixed`, a `due` on the last day of its month stays on the last day (28 February, then 31 March). A `due` on the 29th or 30th clamps in February and then follows the last day. The completion day is the local day of `done_at` in `settings.tz`.
- The successor copies title, notes, project or goal and `repeat_id`. It does not copy attachments, `planned_on` or `slipped`.
- Never more than one open instance per `repeat_id`. An `until` in the past stops the chain.

### Weekly review

- **Quiet:** an open project with open tasks and no task created, updated or done and no edit to the project itself in 21 days, or an open goal with no planned or done task and no edit in 21 days.
- **Surfacing:** Today shows a review link from Friday until it is completed. The review is available any time.
- **Decisions:** `keep` touches the row's `updated_at`, which resets the quiet clock. `pause` sets `paused`, `drop` sets `dropped`. Nothing is deleted.

## 6. Reminders, push, files, calendar

### Reminders and push

- **Scheduler.** A 30-second ticker checks each enabled slot against local time in `settings.tz`. A slot fires when local time is at or past `at_local` and less than 90 minutes later, and no `reminder_log` row exists for that slot and day. Insert the log row first; its primary key prevents a double send.
- **Down for a while.** If the server was off for more than 90 minutes past a slot, skip it for the day.
- **Delivery.** Web Push with VAPID keys generated at first run and kept in the data folder. Payloads encrypted `aes128gcm` (RFC 8291), VAPID JWT per RFC 8292. A 404 or 410 deletes the subscription.
- **Time zone.** One zone for all devices, taken from the browser when the user opens Reminders.

| Slot kind | Title | Body | Opens |
| --- | --- | --- | --- |
| `focus` | One open goal, rotating daily in `sort_key` order | "Today: Easy 5 km run; Fix the import bug." Or "Choose your three for today." | Today |
| `checkin` | "Midday check" | "1 of 2 done. Next: Fix the import bug." Or "All done for today." | Today |
| `wrap` | "Close the day" | "1 of 2 done. Move the rest or let it go." Or "Nothing planned today. Capture what is on your mind." | Today |

### Files

- Stored at `data/files/ab/cd/<sha256>`, never under a user-chosen name.
- Served with `X-Content-Type-Options: nosniff`. Images and PDFs inline with `Content-Security-Policy: default-src 'none'; sandbox`. Everything else as a download.
- A nightly job deletes files with no live attachment older than 7 days.

### Calendar feed

- All-day events (`DTSTART;VALUE=DATE`) for open tasks with a `due` or `planned_on` between 7 days ago and 180 days ahead. A task with both gives two events, titled "Due: ..." and "Plan: ...".
- Stable UIDs (`task-<id>-due@homebase`), `DTSTAMP` from `updated_at`, `REFRESH-INTERVAL` and `X-PUBLISHED-TTL` of one hour, `ETag` and `If-None-Match` supported.
- The URL secret is 32 random bytes. Rotating invalidates the old link.
- Google Calendar polls subscribed feeds only every several hours, so changes appear slowly. Anyone with the link can read task titles.

## 7. Security

- One passphrase, stored only as an argon2id hash in configuration.
- Session token: 32 random bytes in an HttpOnly, Secure, SameSite=Lax cookie. The database stores only its SHA-256.
- The `X-Homebase` header and JSON content type on every write. Strict Content Security Policy, no third-party scripts, self-hosted fonts.
- Every response carries `Content-Security-Policy: default-src 'self'; img-src 'self' data: blob:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'`, `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer` and `X-Frame-Options: DENY`. API responses are `Cache-Control: no-store`.
- Sessions slide: `last_seen` moves at most once an hour, and the cookie is sent again when it does. A session unused for 180 days is deleted.
- The login limit counts per client address. `X-Forwarded-For` is trusted only when the peer is loopback (Caddy or Tailscale on the same machine), and only its last entry.
- Request logs never include query strings, and `/calendar/` paths are logged redacted.
- Backups: `homebase backup <dir>` runs `VACUUM INTO` and copies the files folder.

## 8. Defaults chosen for open decisions

| Decision | Default |
| --- | --- |
| Reminder time zone | One, in settings |
| Task titles in the calendar feed | Real titles behind the secret link |
| Limit of 3 applies to future days | Yes |
| Repeating tasks copy attachments | No |
| Phone reorder control | Long-press drag, plus up and down buttons in the task sheet |
