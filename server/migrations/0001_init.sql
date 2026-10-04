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
