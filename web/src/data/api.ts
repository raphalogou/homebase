// The only module that talks to the server. Components use the hooks in
// this folder instead.

import type {
  AccountInfo,
  Attachment,
  Device,
  Me,
  Op,
  Person,
  Project,
  PullResult,
  PushResult,
  Reminder,
  ReminderInput,
  ReviewAction,
  ReviewSummary,
  ServerSettings,
  SessionInfo,
} from "./types.ts";

/** An error answer from the server, with the code from docs/SPEC.md. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  /** The request field the message belongs to, when the server names one. */
  readonly field: string;

  constructor(status: number, code: string, message: string, field = "") {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.field = field;
  }
}

/** The request never reached the server: offline, or the server is down. */
export class NetworkError extends Error {
  constructor(cause: unknown) {
    super("Cannot reach the server.", { cause });
    this.name = "NetworkError";
  }
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const init: RequestInit = { method, credentials: "same-origin", cache: "no-store" };
  if (method !== "GET") {
    init.headers = { "Content-Type": "application/json", "X-Homebase": "1" };
    init.body = JSON.stringify(body ?? {});
  }
  let res: Response;
  try {
    res = await fetch(path, init);
  } catch (err) {
    throw new NetworkError(err);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  let data: unknown;
  try {
    data = await res.json();
  } catch (err) {
    // A proxy error page or a dropped connection, not our JSON.
    if (!res.ok)
      throw new ApiError(res.status, "unavailable", "The server did not answer properly.");
    throw new NetworkError(err);
  }
  if (!res.ok) {
    const e = (data as { error?: { code?: string; message?: string; field?: string } }).error;
    throw new ApiError(
      res.status,
      e?.code ?? "unknown",
      e?.message ?? "Something went wrong.",
      e?.field ?? "",
    );
  }
  return data as T;
}

async function upload(form: FormData) {
  let res: Response;
  try {
    res = await fetch("/api/files", {
      method: "POST",
      credentials: "same-origin",
      headers: { "X-Homebase": "1" },
      body: form,
    });
  } catch (err) {
    throw new NetworkError(err);
  }
  const data = (await res.json().catch(() => null)) as
    | (Attachment & { error?: undefined })
    | { error?: { code?: string; message?: string } }
    | null;
  if (!res.ok || !data) {
    const e = data?.error;
    throw new ApiError(
      res.status,
      e?.code ?? "unknown",
      e?.message ?? "The upload did not finish.",
    );
  }
  return data as Attachment;
}

export const api = {
  login: (username: string, passphrase: string) =>
    request<void>("POST", "/api/login", { username, passphrase }),
  setupNeeded: async () => (await request<{ needed: boolean }>("GET", "/api/setup")).needed,
  setup: (username: string, passphrase: string) =>
    request<void>("POST", "/api/setup", { username, passphrase }),
  account: () => request<AccountInfo>("GET", "/api/account"),
  changeUsername: (username: string, passphrase: string) =>
    request<AccountInfo>("PUT", "/api/account/username", { username, passphrase }),
  changePassphrase: (current: string, next: string) =>
    request<AccountInfo>("PUT", "/api/account/passphrase", { current, next }),
  logout: () => request<void>("POST", "/api/logout"),
  me: () => request<Me>("GET", "/api/me"),
  pull: (since: number, limit = 500) =>
    request<PullResult>("GET", `/api/sync?since=${since}&limit=${limit}`),
  push: (base: number, ops: Op[]) => request<PushResult>("POST", "/api/sync", { base, ops }),
  /** Uploads a file to a goal, project or task. Fields go first, then the file. */
  upload: (ownerKind: "goal" | "project" | "task", ownerId: string, file: File) => {
    const form = new FormData();
    form.append("ownerKind", ownerKind);
    form.append("ownerId", ownerId);
    form.append("name", file.name);
    form.append("file", file);
    return upload(form);
  },
  /** Type and size of a stored file, from its headers; null if it is gone. */
  fileMeta: async (sha: string): Promise<{ type: string; size: number } | null> => {
    let res: Response;
    try {
      res = await fetch(`/api/files/${sha}`, { method: "HEAD", credentials: "same-origin" });
    } catch (err) {
      throw new NetworkError(err);
    }
    if (!res.ok) return null;
    const disposition = res.headers.get("Content-Disposition") ?? "";
    return {
      type: disposition.startsWith("inline") ? (res.headers.get("Content-Type") ?? "") : "",
      size: Number(res.headers.get("Content-Length") ?? 0),
    };
  },
  pushKey: () => request<{ publicKey: string }>("GET", "/api/push/key"),
  subscribe: (sub: PushSubscriptionJSON, label: string) =>
    request<void>("POST", "/api/push/subscribe", {
      endpoint: sub.endpoint,
      keys: { p256dh: sub.keys?.p256dh, auth: sub.keys?.auth },
      label,
    }),
  unsubscribe: (by: { endpoint: string } | { id: string }) =>
    request<void>("POST", "/api/push/unsubscribe", by),
  devices: () => request<Device[]>("GET", "/api/push/subscriptions"),
  pushTest: () => request<{ sent: number; failed: number }>("POST", "/api/push/test"),
  saveReminders: (list: ReminderInput[]) => request<Reminder[]>("PUT", "/api/reminders", list),
  review: () => request<ReviewSummary>("GET", "/api/review"),
  completeReview: (
    weekStart: string,
    decisions: { kind: "project" | "goal"; id: string; action: ReviewAction }[],
  ) => request<{ rev: number }>("POST", "/api/review/complete", { weekStart, decisions }),
  settings: () => request<ServerSettings>("GET", "/api/settings"),
  rotateCalendar: () => request<{ url: string }>("POST", "/api/calendar/rotate"),
  sessions: () => request<SessionInfo[]>("GET", "/api/sessions"),
  people: () => request<Person[]>("GET", "/api/people"),
  addPerson: (username: string, passphrase: string) =>
    request<Person>("POST", "/api/people", { username, passphrase }),
  removePerson: (id: string) => request<void>("POST", "/api/people/remove", { id }),
  revokeSession: (id: string) => request<void>("POST", "/api/sessions/revoke", { id }),
  putSettings: (tz: string, weekStart: 0 | 1, backups?: boolean) =>
    request<{ tz: string; weekStart: 0 | 1; backups: boolean }>("PUT", "/api/settings", {
      tz,
      weekStart,
      ...(backups === undefined ? {} : { backups }),
    }),
  promote: (taskId: string, goalId: string | null) =>
    request<{ project: Project; removedTaskId: string }>(
      "POST",
      "/api/promote",
      goalId ? { taskId, goalId } : { taskId },
    ),
};

export type Api = typeof api;
