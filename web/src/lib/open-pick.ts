import { useCallback } from "react";
import { useSearchParams } from "react-router";

/** The picker is a modal over the current screen (?pick=1), so Back closes it. */
export function useOpenPick(): [boolean, (open: boolean) => void] {
  const [params, setParams] = useSearchParams();
  const open = params.get("pick") === "1";
  const set = useCallback(
    (next: boolean) => {
      setParams(
        (p) => {
          const copy = new URLSearchParams(p);
          if (next) copy.set("pick", "1");
          else copy.delete("pick");
          return copy;
        },
        { replace: !next },
      );
    },
    [setParams],
  );
  return [open, set];
}
