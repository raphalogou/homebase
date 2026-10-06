# Plan

Build in this order. Do not start a phase before the previous one meets its "Done when". Tick boxes as you go and note surprises at the bottom of the phase. The data model, sync protocol and API are fixed in `docs/SPEC.md`; the look is fixed in `DESIGN.md`.

Status: Phases 0 to 6 done. The owner tested Phase 4's reminders on a real phone over HTTPS on 5 October 2026.

## Decisions already made

- Separate tables for goals, projects and tasks (not one `items` table). Attachments belong to exactly one of them.
- SQLite via `modernc.org/sqlite` (pure Go, no CGO). One binary, one data folder.
- Revision-based sync with row-level last-write-wins. Offline-first client.
- Max 3 tasks per day, and that rule lives on the server.
- Up to 3 daily reminders, written by the server from current data, sent with Web Push.
- Frontend: React, TypeScript, React Router, shadcn/ui on Tailwind, installable PWA.
- Single user, single passphrase. No accounts.

## Phase 0: bootstrap

- [x] Create the layout from `AGENTS.md` (`server/`, `web/`, `deploy/`).
- [x] `server/`: `go mod init`, `cmd/homebase/main.go` serving `/healthz`, config from environment, `slog` logging.
- [x] `web/`: Vite + React + TypeScript (strict), React Router, Tailwind, shadcn/ui initialised, `@fontsource` fonts installed.
- [x] `Makefile` targets: `dev`, `build`, `test`, `lint`.
- [x] CI-equivalent script (`make ci`) that runs the Go tests, `go vet`, the web type check, lint and unit tests.

**Done when:** `make dev` serves the web app with API calls proxied to the Go server, and `make test` passes.

Notes:

- With the owner's approval the stack moved to React 19 and Tailwind v4, because current shadcn/ui components no longer use `forwardRef` and break refs on React 18. Lint and formatting use Biome; web unit tests use `node --test` (Node strips TypeScript types itself), so there is no extra test dependency. Test files import siblings with the `.ts` extension.
- The built web app is embedded from `server/internal/webui/dist`, because `go:embed` cannot reach `web/dist`. `make build` copies it there; the folder is ignored except a placeholder, so a plain `go build` still compiles and answers with a hint.
- `/healthz` and an unknown `/api/*` route (JSON `not_found`) exist. Request logs redact `/calendar/` paths, since the path is the secret.
- Go compiled first time on Go 1.26 with `go 1.22` in `go.mod`.
- `HOMEBASE_PASSPHRASE_HASH` is not read yet; Phase 1 adds it with auth.
- `components.json` names lucide as the shadcn icon library. `DESIGN.md` wants inline SVG icons, so replace lucide imports in added components, or ask before installing `lucide-react`.

## Phase 1: foundation (server)

- [x] Migration runner reading `server/migrations/*.sql` in order, recording `schema_migrations`. `0001_init.sql` copied exactly from `docs/SPEC.md` section 2.
- [x] `store` package behind an interface, so tests can use an in-memory SQLite database. No ORM.
- [x] First-run setup: create the `settings` row with a random calendar token; generate VAPID keys into the data folder.
- [x] Auth: argon2id passphrase check, session cookie, login rate limit, `X-Homebase` header check, `GET /api/me`, logout.
- [x] `GET` and `POST /api/sync` per section 3, including cascade deletes, the 5-minute clock clamp, the `day_full` rule, and server-owned fields.
- [x] Recurrence in the apply step (section 5).
- [x] Daily rollover job, run at 00:05 and on the first sync of a new day.
- [x] `POST /api/promote`.

**Tests required** (table-driven, in-memory DB): conflict resolution both directions; delete versus edit; cascade delete and detach; fourth task on a day; repeat spawn for `fixed` and `after_done`, including month ends; rollover increments `slipped` and clears the plan; promote moves attachments; idempotent re-push of the same ops.

**Done when:** a script that pushes ops from two simulated devices converges to the same state on both, and all tests pass.

Notes:

- The "two devices converge" check is `TestTwoDevicesConverge` in `internal/syncer`: two simulated clients with their own copies and outboxes edit offline, sync in turn, and end up identical to the server. A manual run against the built binary (login, push, the day limit, a repeat successor, paging, promote, logout) also passed.
- Gaps in the spec were filled and written into `docs/SPEC.md` (section 3 "Details", sections 5 and 7). The ones worth a look: dropped tasks do not count toward the three; a delete without `cascade` detaches; the rollover does not touch `updated_at`, so a task ticked done offline at 23:59 is not lost; and rejected ops return the server's copy in `changes`.
- The current `modernc.org/sqlite` and `golang.org/x/crypto` declare `go 1.26.0`, so `go.mod` and the docs now ask for Go 1.26. One database connection serialises all writes, which keeps revisions in commit order.
- Typed errors live in `internal/apperr`; migrations are embedded from `server/migrations` by a small Go file there. `internal/push` only creates the VAPID key so far; `internal/sched` only runs the rollover.
- `homebase hash-passphrase` turns off echo with `stty` rather than adding `golang.org/x/term`.
- Not done, because Phase 1 did not list it: purging tombstones after 90 days.

## Phase 2: daily loop (web)

- [x] Local data layer: IndexedDB mirror of all synced tables, an outbox, and the sync scheduler from section 3. Expose hooks such as `useTasks`, `useToday`. No component talks to `fetch` directly.
- [x] App shell: phone bottom bar (Today, Inbox, Plan, Goals) and desktop left rail, as in `DESIGN.md`.
- [x] Login screen.
- [x] **Today:** goals as quiet lines, Your three (highlighter, rank, done state), capture bar, review link from Friday (moved to Phase 5 with the review itself). Desktop: drag handle reorder. Phone: long-press drag plus up and down buttons in the task sheet.
- [x] **Choose up to three** picker (phone) and the suggestions list (desktop): due soon, in progress, moved a few times.
- [x] **Inbox:** sort with Today, Date, Goal, Project buttons (Project uses `/api/promote`).
- [x] **Plan, Tasks:** week strip, filters (Everything, Due this week, Standalone, Repeating), grouped list, new-task form on desktop.
- [x] Task sheet (phone) and panel (desktop): title, status, due, planned day, goal or project, repeat, notes.
- [x] Service worker via `vite-plugin-pwa` (`injectManifest`), manifest, install prompt, offline shell.

**Done when:** with the network off you can capture, plan three tasks, complete one and reorder; turning the network on syncs everything to a second browser. Lighthouse PWA installability passes.

Notes:

- Verified in headless Firefox, driven over WebDriver BiDi against the built binary: the login flow (including a wrong passphrase); every screen at 390 px and 1280 px; and the "Done when" run. With the server stopped, the app opened from the service worker; a task was captured, one completed, a third planned and the three reordered with the keyboard. After the server came back, the outbox drained, and a second, fresh browser showed the same three in the same order plus the captured task.
- Not verified here: Lighthouse (no Chrome on this machine), long-press drag on a touch screen, mouse dragging of the grip, and a real Android phone. The manifest and worker meet Chrome's installability rules on inspection (name, 192 and 512 px icons, maskable icon, standalone, start URL, a worker with a fetch handler).
- Login screen designed as approved: the same top-aligned column as the other screens.
- UI primitives use Base UI (`@base-ui/react`): Dialog for the bottom sheet, ToggleGroup for the segmented control. Selects are native, which suits phones best. Icons are inline SVG; lucide is not installed.
- The logo is the owner's v2 drawing: a roof over three rounded lines, the first in the highlighter yellow, beside the wordmark at the same height; `web/public/icon.svg` is the source of the PNGs and the notification badge.
- Promote uses `/api/promote` when online with an empty outbox; otherwise it queues the same change as ordinary ops (new project, attachments moved, task deleted), which the server also applies in one transaction.
- Deferred to their phases: the Weekly review and Reminders rail items, Today's review link (Phase 5), and the Tasks/Projects switch on Plan (Phase 3). Goal creation has no screen yet (Phase 6), so the Inbox Goal button only lists existing goals.
- The service worker serves the cached shell first, so a new build shows on the second open after a deploy.
- A failing install was caught in testing: the plugin listed the manifest and icons twice, and `cache.addAll` rejects duplicates. The worker now removes duplicates itself.

## Phase 3: goals, projects, attachments

- [x] Goals list and goal detail (projects, tasks, notes, links).
- [x] Plan, Projects: grouped by goal, add a project.
- [x] Project detail: tasks, notes (saved on pause), attachments.
- [x] Link and note attachments through sync.
- [x] `POST /api/files`, `GET /api/files/{sha}`, upload UI (button and desktop drop area), orphan clean-up job, serving headers from section 6.
- [x] Delete flows that ask: delete with contents or keep tasks standalone.

**Tests required:** upload size limit; same file twice stores once; hostile file served as download with `nosniff`; attachment owner check (exactly one owner).

**Done when:** a project with notes, a link, a note and two files survives a reload and shows on a second device.

Notes:

- Verified in headless Chromium (ungoogled-chromium 153, over the DevTools protocol) against the built binary. A goal was created through the new field, then a project under it, two tasks, notes, a link, a note, and a PDF and a PNG uploaded through the file input. All of it survived a reload and appeared on a second, fresh browser; the PDF downloaded there with its type. Chrome reported no installability errors (`Page.getInstallabilityErrors`), which also covers the Phase 2 check that Lighthouse no longer offers.
- Approved designs, now recorded in `DESIGN.md`: the "New goal" field under the Goals list (pulled forward from Phase 6) and the delete question as a sheet or dialog.
- Uploads need a connection; offline the button says so. Links and notes work offline like any other row. Before uploading to a goal or project made offline, the outbox is sent first so the server knows the owner.
- File types and sizes in attachment rows come from a `HEAD` request, because the attachment row carries neither. Offline they show as the file's extension.
- Deleting shows the server's cascade at once on the device; the server's own rows replace it at the next sync.
- Found in testing: an add field cleared itself after saving, wiping anything typed meanwhile. Fields now clear on submit.

Changes after review (approved by the owner): toasts with Undo and Open; the picker as a modal; a week strip with previous and next weeks and clickable days; icons with counts on the Projects list; clearer Inbox verbs; Lucide for all icons; "Change" and "Remove from today" to take tasks off today; regions that fill the window. Found while testing: Base UI keeps toasts past the limit (marked `limited`) unless the renderer skips them; promote now shows its result at once instead of waiting for a sync.

## Phase 4: reminders and push

- [x] `GET /api/push/key`, subscribe, unsubscribe, test.
- [x] Web Push sender implemented to RFC 8291 and RFC 8292. Use the standard library plus `golang.org/x/crypto` only if needed. Include a unit test that encrypts a payload and decrypts it with the subscriber's key.
- [x] Scheduler: 30-second ticker, `reminder_log` insert-first guard, 90-minute grace, message builders from section 6.
- [x] Service worker `push` and `notificationclick` handlers; tapping opens Today and triggers a sync.
- [x] Reminders screen (phone and desktop): three slots, toggles, time inputs, device list, test button, notification permission flow.

**Done when:** on an Android phone with the app installed and closed, the 08:00 notification arrives once and tapping it opens Today. Deploy first (HTTPS required) to test this.

Notes:

- Web Push is written with the standard library only (`crypto/ecdh`, `crypto/hkdf`, AES-GCM, ES256), so no module was added. The encryption is checked byte for byte against the worked example in RFC 8291 Appendix A, and against a decrypting test browser. The VAPID token is checked by verifying its signature. A test push service over TLS checks the whole path, including removing a subscription the service reports gone (410).
- Approved and added to the spec: `GET /api/push/subscriptions` (devices without their endpoints) and `POST /api/push/unsubscribe` by `id`, so an old phone can be removed from the laptop.
- Verified in headless Chromium: the Reminders screen at both widths; saving a time and a toggle (they reach the server and other devices through sync); the browser's zone adopted on opening Reminders; the service worker showing a reminder fed through the DevTools protocol, with title, body, tag, icon and the status-bar badge. Ungoogled Chromium has no push service, so turning reminders on there shows "This browser cannot get reminders."
- Verified by the owner on a real phone over HTTPS (5 October 2026): the phone setup for reminders works, which the "Done when" needed.
- `notificationclick` uses the open window if there is one, otherwise opens `/`. It only ever opens paths of this app.
- Found while testing: a failed browser subscribe threw a browser error the screen did not show; it now explains itself. Long toast messages wrap instead of being cut off.

## Phase 5: finish

- [x] Recurrence UI (repeat picker: every N days, weeks, months; fixed or after done; end date).
- [x] Weekly review (`/api/review`, `/api/review/complete`), phone and desktop.
- [x] Calendar feed and rotate button.
- [x] `homebase backup <dir>` command and restore note in the README.
- [x] Deploy files in `deploy/`: Dockerfile, Caddyfile, systemd unit.
- [x] Settings screen: time zone, week start, calendar link, sessions list.

**Done when:** a fresh install on a clean server follows the README top to bottom and works.

Notes:

- **Done when, checked:** the README's Docker steps were followed as written on this machine, changing only the names (`homebase-test`) and the port. Build the image, write the settings file with `hash-passphrase`, start it, check `/healthz`, log in. Then the backup steps (recreate with the folder mounted, `docker exec … homebase backup`) and the restore (stop, copy the backup over the database, start), which brought back the state of the backup. The runtime image is `busybox:musl` (owner's choice over distroless and Alpine): 22.6 MB, with `sh`, `stty` and `wget`, and the CA bundle copied from the build stage. Not followed: the systemd path (needs root on a server) and HTTPS with Caddy or Tailscale (needs a domain).
- Found while installing: Docker's `--env-file` keeps the single quotes that `hash-passphrase` prints (systemd and shells strip them). The server now removes one pair of matching quotes, so both work.
- Approved and added to the spec: `GET /api/sessions` and `POST /api/sessions/revoke`. Small additions the approved screens need: `completed` in the review and `calendarUrl` in the settings.
- Changed rule: opening Reminders adopts the browser's zone only while it is still the first-run UTC, so a zone chosen in Settings sticks.
- Verified in headless Chromium against the Docker install: the review link on Today (a Sunday, with Monday weeks), the review with a quiet project and goal, pausing the goal (it left Today's goals) and the link gone after "Finish review"; the repeat picker ("Every week on Monday and Thursday, from the due date."); Settings with the calendar link, a new link replacing it, and the sessions list; the rail order.
- The quick "Repeat" select stays in Plan's "New task" form; the full picker is in the task sheet and panel.
- Backups keep every database copy; pruning old ones is left to the owner (the README says so).

## Phase 6: polish

Designed by the owner (the Phase 6 addendum, now merged into `DESIGN.md`).

- [x] Account: a username besides the passphrase, a first-run Set up screen, and changing either from Settings (`0002_account.sql`, `docs/SPEC.md` sections 2, 4 and 7).
- [x] Dark theme, with the light contrast fixes (`--muted`, the new `--border`).
- [x] Empty, loading and error states for every screen.
- [x] Offline and sync-failed indicator, phone and desktop.
- [x] Phone Projects list, goal creation, and the repeat picker. (Done in Phases 3 and 5.)
- [x] Keyboard shortcuts on desktop (capture, switch screens).
- [x] Accessibility pass: focus order, screen reader labels, reduced motion.

Notes:

- **To confirm with the owner:** an install from before usernames gets its account from `HOMEBASE_PASSPHRASE_HASH` with the username `owner`, so nobody is locked out; it can be renamed in Settings. The variable is ignored once the account exists, so a passphrase changed in Settings survives a restart. Without the variable, a fresh install shows Set up, and whoever reaches it first creates the account; the README says to do it straight after starting.
- Errors from the account endpoints carry a `field`, so each form shows the message under the right input. A refused login always names the passphrase field and never says which part was wrong. The change endpoints share the five-per-ten-minutes login limit.
- `GET /api/me` now includes the username, so Log in can fill it in after a session ends, even offline. A session that ended keeps the local copy and its outbox; an explicit logout still empties them.
- The theme choice is per device and must be read before the app's code loads, so it lives in `localStorage` (the only synchronous store) and is applied by `public/theme.js`. An inline script would be simpler but the Content-Security-Policy forbids it.
- Base UI toasts are non-modal dialogs (`role="dialog"`, `aria-modal="false"`); the shortcut handler only stands down for modal ones.
- "Loading" means the first sync after logging in has not finished (a `loaded` flag in IndexedDB). Devices that synced before the flag existed count as loaded.
- The phone's Plan gained the plus button the mockups draw, opening the existing New task form in a sheet; without it the phone could not add a dated task from Plan.
- Verified in headless Chromium against the built binary, at 390 and 1280 px: Set up with its errors, Log in, a wrong pair, the session-ended notice, both change dialogs (a wrong current passphrase, then success, after which Sessions listed only this browser), dark and light themes, the offline and failed-sync banners with Retry and "Back online. Synced.", every empty state on a fresh install, the skeleton while the first sync is held back, the error block when it fails and "Try again", a project that no longer exists, a 26 MB upload, and every shortcut. A script over every screen in both themes found no unnamed buttons, unlabelled fields, targets under 44 px or text under 4.5:1. Not verified: a real screen reader, and a real phone.

## After release: several people (owner's request, 6 October 2026)

- [x] A planner per person in `data/users/<id>/`, the existing schema unchanged apart from `0004_users.sql`; an install from before moves into the owner's folder on start, keeping its sessions.
- [x] Cookies name their planner; login finds the person by username, with one limit across everyone and an unknown username timed like a wrong passphrase.
- [x] The owner adds people with a one-time passphrase and removes them (their folder moves to `removed/`); usernames are unique across everyone.
- [x] "Choose your passphrase" covers the app until an added person replaces the one-time passphrase.
- [x] Everything else is per person: settings, theme, calendar link, reminders, sessions, the backup switch (each into `<backup folder>/<id>`).
- [x] A device that held someone else's planner empties itself before syncing the new person, so nothing crosses between accounts.
- Verified: Go tests for each rule above; in headless Chromium against a copy of a v0.1.0 install (owner and a task), the move, the old cookie, adding Sam, Sam's first login and passphrase, Sam not seeing the owner's task, Sam without the People tab, removing Sam, at 390 and 1280 px.

## Known risks

- Push needs HTTPS and an installed app on Android; test early on a real device.
- Row-level last-write-wins can drop one of two near-simultaneous edits of the same row. Accepted for one user.
- Google Calendar refreshes subscribed feeds only every several hours.
- The first Go build was never run in the sandbox this plan was written in (modules could not be downloaded), so expect small compile fixes when starting Phase 1.
