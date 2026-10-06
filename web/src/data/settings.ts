// Settings: zone and week start, the calendar link, backups, the account,
// sessions, and for the owner the people on this server.

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

export function useAccount() {
  return useOnline(api.account);
}

export function usePeople() {
  return useOnline(api.people);
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
    saveBackups: useCallback(
      async (on: boolean) => {
        const me = store.me;
        await needsServer(
          api.putSettings(me?.tz ?? "UTC", me?.weekStart ?? 1, on),
          "Changing backups",
        );
      },
      [store],
    ),
    rotateCalendar: useCallback(
      async () => (await needsServer(api.rotateCalendar(), "Making a new link")).url,
      [],
    ),
    /**
     * The account changes keep the server's ApiError, whose field tells the
     * form which input the message belongs under.
     */
    changeUsername: useCallback(
      async (username: string, passphrase: string) => {
        const info = await api.changeUsername(username, passphrase);
        if (store.me) await store.setMe({ ...store.me, username: info.username });
        return info;
      },
      [store],
    ),
    changePassphrase: useCallback(
      async (current: string, next: string) => {
        const info = await api.changePassphrase(current, next);
        if (store.me) await store.setMe({ ...store.me, mustChange: false });
        return info;
      },
      [store],
    ),
    addPerson: useCallback(
      (username: string, passphrase: string) => api.addPerson(username, passphrase),
      [],
    ),
    removePerson: useCallback(
      (id: string) => needsServer(api.removePerson(id), "Removing a person"),
      [],
    ),
    revoke: useCallback(
      (id: string) => needsServer(api.revokeSession(id), "Logging out a session"),
      [],
    ),
  };
}
