/// <reference lib="webworker" />
// Service worker: keeps the app shell offline. Push and notificationclick
// handlers arrive with reminders (PLAN.md Phase 4).

// A module, so this `self` shadows the DOM's global one.
export type {};

declare const self: ServiceWorkerGlobalScope & {
  __WB_MANIFEST: (string | { url: string; revision: string | null })[];
};

// Unique by URL: cache.addAll rejects a list with duplicates, which would
// make every install fail.
const manifest = [
  ...new Map(
    self.__WB_MANIFEST.map((e) => {
      const entry = typeof e === "string" ? { url: e, revision: null } : e;
      return [entry.url, entry] as const;
    }),
  ).values(),
];

// One cache per build: the name changes whenever any precached file does.
function hash(s: string): string {
  let h = 5381;
  for (let i = 0; i < s.length; i++) h = ((h << 5) + h + s.charCodeAt(i)) | 0;
  return (h >>> 0).toString(36);
}
const CACHE = `shell-${hash(JSON.stringify(manifest))}`;
const SHELL = "/index.html";

// Never cached: data, the calendar feed and the health check go to the network.
function bypass(url: URL): boolean {
  return (
    url.origin !== self.location.origin ||
    url.pathname.startsWith("/api/") ||
    url.pathname.startsWith("/calendar/") ||
    url.pathname === "/healthz"
  );
}

self.addEventListener("install", (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(CACHE);
      await cache.addAll(manifest.map((e) => new Request(e.url, { cache: "reload" })));
      await self.skipWaiting();
    })(),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    (async () => {
      for (const name of await caches.keys()) {
        if (name !== CACHE) await caches.delete(name);
      }
      await self.clients.claim();
    })(),
  );
});

self.addEventListener("fetch", (event) => {
  const req = event.request;
  if (req.method !== "GET") return;
  const url = new URL(req.url);
  if (bypass(url)) return;

  // Every page is the same single-page app, so navigations get the shell,
  // from the cache first: the app opens instantly and offline.
  if (req.mode === "navigate") {
    event.respondWith(
      (async () => (await caches.match(SHELL, { cacheName: CACHE })) ?? fetch(req))(),
    );
    return;
  }

  event.respondWith((async () => (await caches.match(req, { cacheName: CACHE })) ?? fetch(req))());
});

interface PushMessage {
  title: string;
  body: string;
  url: string;
  tag: string;
  sync: boolean;
}

// A reminder from the server (docs/SPEC.md section 6). The message also asks
// any open window to sync, so the app shows what the notification says.
self.addEventListener("push", (event) => {
  let msg: PushMessage = { title: "Homebase", body: "", url: "/", tag: "", sync: false };
  try {
    msg = { ...msg, ...(event.data?.json() as Partial<PushMessage>) };
  } catch {
    // An unreadable payload still shows the plain fallback.
  }
  event.waitUntil(
    (async () => {
      const options: NotificationOptions = {
        body: msg.body,
        icon: "/icon-192.png",
        badge: "/badge-96.png",
        data: { url: msg.url },
      };
      if (msg.tag) options.tag = msg.tag;
      await self.registration.showNotification(msg.title, options);
      if (msg.sync) {
        for (const c of await self.clients.matchAll({ type: "window" }))
          c.postMessage({ type: "sync" });
      }
    })(),
  );
});

// Tapping a reminder opens Today, in the window already open if there is one.
self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const wanted = (event.notification.data as { url?: string } | null)?.url ?? "/";
  // Only ever a path of this app, never another site.
  const url = new URL(wanted.startsWith("/") ? wanted : "/", self.location.origin).href;
  event.waitUntil(
    (async () => {
      const windows = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
      const open = windows[0];
      if (open) {
        const client = await open.focus();
        await client.navigate(url).catch(() => null);
        client.postMessage({ type: "sync" });
        return;
      }
      await self.clients.openWindow(url);
    })(),
  );
});
