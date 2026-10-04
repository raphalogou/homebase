import { useCallback, useMemo, useState } from "react";

/**
 * When a form shows its errors: a field's own check only after the person
 * leaves it or presses the button, and the server's answer until that field
 * is edited again. What was typed is never cleared.
 */
export function useFieldErrors<F extends string>() {
  const [left, setLeft] = useState<ReadonlySet<F> | "all">(new Set());
  const [server, setServer] = useState<Partial<Record<F, string>>>({});

  const blur = useCallback(
    (f: F) => setLeft((s) => (s === "all" || s.has(f) ? s : new Set(s).add(f))),
    [],
  );
  const submitted = useCallback(() => setLeft("all"), []);
  const edited = useCallback(
    (f: F) => setServer((s) => (s[f] === undefined ? s : { ...s, [f]: undefined })),
    [],
  );
  const fromServer = useCallback(
    (f: F, message: string) => setServer({ [f]: message } as Partial<Record<F, string>>),
    [],
  );
  const clear = useCallback(() => {
    setLeft(new Set());
    setServer({});
  }, []);

  return useMemo(
    () => ({
      blur,
      submitted,
      edited,
      fromServer,
      clear,
      /** The message to show under f, given its own check's result. */
      error: (f: F, local: string): string =>
        server[f] || (left === "all" || left.has(f) ? local : ""),
    }),
    [blur, submitted, edited, fromServer, clear, server, left],
  );
}
