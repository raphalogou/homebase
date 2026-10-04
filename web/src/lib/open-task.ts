import { useCallback } from "react";
import { useSearchParams } from "react-router";

/** The open task lives in the URL (?task=ID), so Back closes it. */
export function useOpenTask(): [string | null, (id: string | null) => void] {
  const [params, setParams] = useSearchParams();
  const id = params.get("task");
  const open = useCallback(
    (next: string | null) => {
      setParams(
        (p) => {
          const copy = new URLSearchParams(p);
          if (next) copy.set("task", next);
          else copy.delete("task");
          return copy;
        },
        { replace: next === null },
      );
    },
    [setParams],
  );
  return [id, open];
}
