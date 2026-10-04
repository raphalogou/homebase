// Settings: zone and week start, the calendar link, and sessions.

import { useCallback } from "react";
import { api } from "./api.ts";
import { needsServer, useOnline } from "./online.ts";
import { useData } from "./provider.tsx";

export function useServerSettings() {
  return useOnline(api.settings);
}

export function useSessions() {
  return useOnline(api.sessions);
}

export function useSettingsActions() {
  const { store } = useData();
  return {
    saveTime: useCallback(
      async (tz: string, weekStart: 0 | 1) => {
        const s = await needsServer(api.putSettings(tz, weekStart), "Changing settings");
        if (store.me) await store.setMe({ ...store.me, tz: s.tz, weekStart: s.weekStart });
      },
      [store],
    ),
    rotateCalendar: useCallback(
      async () => (await needsServer(api.rotateCalendar(), "Making a new link")).url,
      [],
    ),
    revoke: useCallback(
      (id: string) => needsServer(api.revokeSession(id), "Logging out a session"),
      [],
    ),
  };
}
