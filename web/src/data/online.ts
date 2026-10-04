// Screens that read server-computed data (the review, settings, sessions)
// load it on open. They need a connection, unlike the synced rows.

import { useCallback, useEffect, useState } from "react";
import { ApiError, NetworkError } from "./api.ts";
import { ActionError } from "./hooks.ts";

export type Loaded<T> =
  | { state: "loading" }
  | { state: "offline" }
  | { state: "error"; message: string }
  | { state: "ready"; data: T };

/** Loads once on mount; reload() fetches again. */
export function useOnline<T>(load: () => Promise<T>): [Loaded<T>, () => void] {
  const [value, setValue] = useState<Loaded<T>>({ state: "loading" });
  const [n, setN] = useState(0);
  // biome-ignore lint/correctness/useExhaustiveDependencies: n asks for a fresh load
  useEffect(() => {
    let live = true;
    load().then(
      (data) => live && setValue({ state: "ready", data }),
      (err: unknown) => {
        if (!live) return;
        if (err instanceof NetworkError) setValue({ state: "offline" });
        else
          setValue({
            state: "error",
            message: err instanceof Error ? err.message : "Something went wrong.",
          });
      },
    );
    return () => {
      live = false;
    };
  }, [n]);
  return [value, useCallback(() => setN((x) => x + 1), [])];
}

/** Turns API failures into messages a toast can show. */
export function needsServer<T>(p: Promise<T>, what: string): Promise<T> {
  return p.catch((err: unknown) => {
    if (err instanceof NetworkError) throw new ActionError(`${what} needs a connection.`);
    if (err instanceof ApiError) throw new ActionError(err.message);
    throw err;
  });
}
