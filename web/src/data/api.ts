// The only module that talks to the server. Components use the hooks in
// this folder instead.

import type { Me, Op, Project, PullResult, PushResult } from "./types.ts";

/** An error answer from the server, with the code from docs/SPEC.md. */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
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
    const e = (data as { error?: { code?: string; message?: string } }).error;
    throw new ApiError(res.status, e?.code ?? "unknown", e?.message ?? "Something went wrong.");
  }
  return data as T;
}

export const api = {
  login: (passphrase: string) => request<void>("POST", "/api/login", { passphrase }),
  logout: () => request<void>("POST", "/api/logout"),
  me: () => request<Me>("GET", "/api/me"),
  pull: (since: number, limit = 500) =>
    request<PullResult>("GET", `/api/sync?since=${since}&limit=${limit}`),
  push: (base: number, ops: Op[]) => request<PushResult>("POST", "/api/sync", { base, ops }),
  promote: (taskId: string, goalId: string | null) =>
    request<{ project: Project; removedTaskId: string }>(
      "POST",
      "/api/promote",
      goalId ? { taskId, goalId } : { taskId },
    ),
};

export type Api = typeof api;
