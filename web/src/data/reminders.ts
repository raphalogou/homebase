// Reminders and this device's Web Push subscription.

import { useCallback, useEffect, useMemo, useState } from "react";
import { labelFromUA } from "../lib/ua.ts";
import { ApiError, api, NetworkError } from "./api.ts";
import { ActionError, useVersion } from "./hooks.ts";
import { useData } from "./provider.tsx";
import type { Device, Reminder, ReminderInput } from "./types.ts";

/** The three slots, as synced rows, in slot order. */
export function useReminders(): Reminder[] {
  const { store } = useData();
  const v = useVersion();
  // biome-ignore lint/correctness/useExhaustiveDependencies: v is the store's change counter
  return useMemo(() => [...store.reminders.values()].sort((a, b) => a.slot - b.slot), [store, v]);
}

function online<T>(p: Promise<T>): Promise<T> {
  return p.catch((err: unknown) => {
    if (err instanceof NetworkError)
      throw new ActionError("Changing reminders needs a connection.");
    if (err instanceof ApiError) throw new ActionError(err.message);
    throw err;
  });
}

/** Saves one slot; the server owns reminder rows, so this needs a connection. */
export function useSaveReminder(): (r: ReminderInput) => Promise<void> {
  const { store } = useData();
  return useCallback(
    async (r: ReminderInput) => {
      const rows = await online(api.saveReminders([r]));
      await store.apply({
        goals: [],
        projects: [],
        tasks: [],
        repeats: [],
        attachments: [],
        reminders: rows,
      });
    },
    [store],
  );
}

/**
 * Reminder times follow one zone, taken from the browser when the user opens
 * Reminders (docs/SPEC.md section 6) while it is still the first-run UTC.
 * After that, Settings is where it changes, so a choice made there sticks.
 * Returns the zone in use.
 */
export function useAdoptBrowserZone(): string | null {
  const { store } = useData();
  useVersion();
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
  useEffect(() => {
    const me = store.me;
    if (!me || !zone || me.tz === zone || me.tz !== "UTC") return;
    api.putSettings(zone, me.weekStart).then(
      (s) => store.setMe({ ...me, tz: s.tz, weekStart: s.weekStart }),
      () => {},
    );
  }, [store, zone]);
  return store.me?.tz ?? null;
}

export type PushState =
  | { kind: "loading" }
  | { kind: "unsupported" }
  | { kind: "blocked" }
  | { kind: "off" }
  | { kind: "on"; id: string };

/** The id the server gives a subscription: the endpoint's SHA-256, first 16 bytes, hex. */
async function deviceId(endpoint: string): Promise<string> {
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(endpoint));
  return [...new Uint8Array(digest).slice(0, 16)]
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

function keyBytes(base64url: string): Uint8Array<ArrayBuffer> {
  const b64 = base64url.replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(b64 + "=".repeat((4 - (b64.length % 4)) % 4));
  return Uint8Array.from(raw, (c) => c.charCodeAt(0));
}

async function registration(): Promise<ServiceWorkerRegistration | null> {
  if (!("serviceWorker" in navigator)) return null;
  return (await navigator.serviceWorker.getRegistration()) ?? null;
}

/** Whether this device gets reminders, and the other devices that do. */
export function usePush() {
  const [state, setState] = useState<PushState>({ kind: "loading" });
  const [devices, setDevices] = useState<Device[] | null>(null);

  const refresh = useCallback(async () => {
    const supported =
      "PushManager" in window && "Notification" in window && "serviceWorker" in navigator;
    if (!supported) {
      setState({ kind: "unsupported" });
    } else if (Notification.permission === "denied") {
      setState({ kind: "blocked" });
    } else {
      const reg = await registration();
      const sub = await reg?.pushManager.getSubscription();
      setState(sub ? { kind: "on", id: await deviceId(sub.endpoint) } : { kind: "off" });
    }
    api.devices().then(setDevices, () => setDevices(null));
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const turnOn = useCallback(async () => {
    const permission = await Notification.requestPermission();
    if (permission !== "granted") {
      await refresh();
      throw new ActionError(
        permission === "denied"
          ? "Notifications are blocked. Allow them in the browser's site settings."
          : "Reminders stay off until you allow notifications.",
      );
    }
    const reg = await registration();
    if (!reg) throw new ActionError("Install the app or reload the page, then try again.");
    const { publicKey } = await online(api.pushKey());
    let sub: PushSubscription;
    try {
      sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: keyBytes(publicKey),
      });
    } catch {
      // Browsers without a push service (or with notifications blocked for
      // the site) refuse here; say so rather than fail silently.
      await refresh();
      throw new ActionError("This browser cannot get reminders. Try Chrome on Android.");
    }
    await online(api.subscribe(sub.toJSON(), labelFromUA(navigator.userAgent)));
    await refresh();
  }, [refresh]);

  const turnOff = useCallback(async () => {
    const reg = await registration();
    const sub = await reg?.pushManager.getSubscription();
    if (sub) {
      await online(api.unsubscribe({ endpoint: sub.endpoint }));
      await sub.unsubscribe();
    }
    await refresh();
  }, [refresh]);

  const remove = useCallback(
    async (id: string) => {
      await online(api.unsubscribe({ id }));
      await refresh();
    },
    [refresh],
  );

  const sendTest = useCallback(() => online(api.pushTest()), []);

  return { state, devices, turnOn, turnOff, remove, sendTest };
}
