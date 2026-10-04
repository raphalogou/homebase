import { useCallback } from "react";
import { useSearchParams } from "react-router";

/**
 * A modal over the current screen, kept in the URL (?pick=1, ?settings=1)
 * so Back closes it. Opening or switching view adds a history entry;
 * closing replaces it.
 */
export function useModal(
  name: string,
): [string | null, (view: string | null, replace?: boolean) => void] {
  const [params, setParams] = useSearchParams();
  const set = useCallback(
    (view: string | null, replace = !view) => {
      setParams(
        (p) => {
          const copy = new URLSearchParams(p);
          if (view) copy.set(name, view);
          else copy.delete(name);
          return copy;
        },
        { replace },
      );
    },
    [name, setParams],
  );
  return [params.get(name), set];
}

/** Opens a modal by name, for places that only open one (the rail's key, Today's links). */
export function useOpenModal(): (name: string) => void {
  const [, setParams] = useSearchParams();
  return useCallback(
    (name: string) =>
      setParams((p) => {
        const copy = new URLSearchParams(p);
        copy.set(name, "1");
        return copy;
      }),
    [setParams],
  );
}

export function useOpenPick(): [boolean, (open: boolean) => void] {
  const [view, set] = useModal("pick");
  return [view === "1", useCallback((open: boolean) => set(open ? "1" : null), [set])];
}
