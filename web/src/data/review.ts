// The weekly review is computed by the server, so it needs a connection.

import { useCallback } from "react";
import { weekday } from "../lib/dates.ts";
import { api } from "./api.ts";
import { needsServer, useOnline } from "./online.ts";
import { useData } from "./provider.tsx";
import type { ReviewAction, ReviewSummary } from "./types.ts";

export function useReview() {
  return useOnline(api.review);
}

/** Sends the decisions, then syncs so paused and dropped rows show at once. */
export function useCompleteReview() {
  const { engine } = useData();
  return useCallback(
    async (s: ReviewSummary, choices: Record<string, ReviewAction>) => {
      await needsServer(
        api.completeReview(
          s.weekStart,
          s.quiet.map((q) => ({ kind: q.kind, id: q.id, action: choices[q.id] ?? "keep" })),
        ),
        "Finishing the review",
      );
      await engine.run();
    },
    [engine],
  );
}

/**
 * Whether Today should offer the review: from Friday to the end of the week,
 * until this week's review is done (docs/SPEC.md section 5). Before Friday
 * it does not ask the server at all.
 */
export function useReviewDue(today: string, weekStart: 0 | 1): boolean {
  const fridayIndex = (5 - weekStart + 7) % 7;
  const index = (weekday(today) - weekStart + 7) % 7;
  const inWindow = index >= fridayIndex;
  const [review] = useOnline(inWindow ? api.review : notYet);
  return inWindow && review.state === "ready" && review.data !== null && !review.data.completed;
}

const notYet = (): Promise<null> => Promise.resolve(null);
