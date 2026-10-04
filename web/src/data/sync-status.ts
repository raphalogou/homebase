// What the offline and sync indicator shows (DESIGN.md "Offline and sync
// indicator"): nothing while all is well, a banner while offline or failing,
// "Syncing" only once a sync has taken a second, and "Back online. Synced."
// for three seconds after trouble clears.

import { useEffect, useState, useSyncExternalStore } from "react";
import { useVersion } from "./hooks.ts";
import { useData } from "./provider.tsx";
import { indicatorFor, type SyncIndicator } from "./sync-indicator.ts";

const SLOW_MS = 1000;
const RECOVERED_MS = 3000;

export function useSyncIndicator(): { indicator: SyncIndicator; retry: () => void } {
  const { engine, store } = useData();
  useVersion();
  const state = useSyncExternalStore(engine.onState, () => engine.state);
  const retryAt = useSyncExternalStore(engine.onState, () => engine.retryAt);
  const [slow, setSlow] = useState(false);
  const [trouble, setTrouble] = useState<"offline" | "error" | null>(null);
  const [recovered, setRecovered] = useState(false);
  const [now, setNow] = useState(Date.now);

  useEffect(() => {
    setSlow(false);
    if (state !== "syncing") return;
    const t = setTimeout(() => setSlow(true), SLOW_MS);
    return () => clearTimeout(t);
  }, [state]);

  useEffect(() => {
    if (state === "offline" || state === "error") setTrouble(state);
    else if (state === "idle" && trouble) {
      setTrouble(null);
      setRecovered(true);
    }
  }, [state, trouble]);

  useEffect(() => {
    if (!recovered) return;
    const t = setTimeout(() => setRecovered(false), RECOVERED_MS);
    return () => clearTimeout(t);
  }, [recovered]);

  // The countdown to the next try moves once a second while it shows.
  useEffect(() => {
    if (state !== "error") return;
    setNow(Date.now());
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, [state]);

  return {
    indicator: indicatorFor({
      state,
      slow,
      trouble,
      recovered,
      waiting: store.outbox.size,
      retryIn: Math.max(0, Math.ceil((retryAt - now) / 1000)),
    }),
    retry: () => void engine.run(),
  };
}

/** True until this device has finished its first sync since logging in. */
export function useFirstLoad(): { loading: boolean; failed: boolean; retry: () => void } {
  const { engine, store } = useData();
  useVersion();
  const state = useSyncExternalStore(engine.onState, () => engine.state);
  return {
    loading: !store.loaded,
    failed: !store.loaded && (state === "offline" || state === "error"),
    retry: () => void engine.run(),
  };
}
