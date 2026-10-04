// Row shapes as the server sends them (docs/SPEC.md section 3, "Details").

export type GoalStatus = "open" | "paused" | "done" | "dropped";
export type TaskStatus = "open" | "done" | "dropped";

/** A local calendar day, "YYYY-MM-DD". */
export type LocalDate = string;

export interface Goal {
  id: string;
  title: string;
  notes: string;
  status: GoalStatus;
  targetDate: LocalDate | null;
  sortKey: number;
  createdAt: number;
  updatedAt: number;
  deletedAt: number | null;
  rev: number;
}

export interface Project {
  id: string;
  goalId: string | null;
  title: string;
  notes: string;
  status: GoalStatus;
  due: LocalDate | null;
  createdAt: number;
  updatedAt: number;
  deletedAt: number | null;
  rev: number;
}

export interface Repeat {
  id: string;
  freq: "day" | "week" | "month";
  every: number;
  weekdays: number | null;
  mode: "fixed" | "after_done";
  until: LocalDate | null;
  createdAt: number;
  updatedAt: number;
  deletedAt: number | null;
  rev: number;
}

export interface Task {
  id: string;
  projectId: string | null;
  goalId: string | null;
  title: string;
  notes: string;
  status: TaskStatus;
  due: LocalDate | null;
  plannedOn: LocalDate | null;
  planRank: number | null;
  slipped: number;
  repeatId: string | null;
  doneAt: number | null;
  createdAt: number;
  updatedAt: number;
  deletedAt: number | null;
  rev: number;
}

export interface Attachment {
  id: string;
  goalId: string | null;
  projectId: string | null;
  taskId: string | null;
  kind: "link" | "note" | "file";
  name: string;
  url: string | null;
  body: string | null;
  fileSha: string | null;
  createdAt: number;
  updatedAt: number;
  deletedAt: number | null;
  rev: number;
}

export interface Reminder {
  slot: number;
  enabled: boolean;
  atLocal: string;
  kind: "focus" | "checkin" | "wrap";
  updatedAt: number;
  rev: number;
}

export interface Changes {
  goals: Goal[];
  projects: Project[];
  tasks: Task[];
  repeats: Repeat[];
  attachments: Attachment[];
  reminders: Reminder[];
}

/** Tables a client may write through sync. */
export type SyncTable = "goals" | "projects" | "tasks" | "repeats" | "attachments";

export interface RowOf {
  goals: Goal;
  projects: Project;
  tasks: Task;
  repeats: Repeat;
  attachments: Attachment;
}

export interface Op {
  op: "upsert" | "delete";
  table: SyncTable;
  id: string;
  row?: unknown;
  updatedAt: number;
  cascade?: "delete" | "detach";
}

export interface PushResult {
  rev: number;
  applied: string[];
  rejected: { id: string; reason: string }[];
  changes: Changes;
}

export interface PullResult extends Changes {
  rev: number;
  more: boolean;
}

export interface Me {
  tz: string;
  weekStart: 0 | 1;
  rev: number;
}

export type ReminderInput = Pick<Reminder, "slot" | "enabled" | "atLocal" | "kind">;

/** A device that gets reminders; its push endpoint never leaves the server. */
export interface Device {
  id: string;
  label: string;
  createdAt: number;
  lastOk: number | null;
}

export interface ReviewSummary {
  weekStart: LocalDate;
  doneCount: number;
  doneByGoal: { goalId: string; title: string; count: number }[];
  quiet: { kind: "project" | "goal"; id: string; title: string; lastActivity: number }[];
  goalsWithNothing: { id: string; title: string }[];
  completed: boolean;
}

export type ReviewAction = "keep" | "pause" | "drop";

export interface ServerSettings {
  tz: string;
  weekStart: 0 | 1;
  calendarUrl: string;
}

export interface SessionInfo {
  id: string;
  label: string;
  createdAt: number;
  lastSeen: number;
  current: boolean;
}
