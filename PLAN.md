# Plan

Build in this order. Do not start a phase before the previous one meets its "Done when". Tick boxes as you go and note surprises at the bottom of the phase. The data model, sync protocol and API are fixed in `docs/SPEC.md`; the look is fixed in `DESIGN.md`.

Status: Phase 0 done.

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

- [ ] Migration runner reading `server/migrations/*.sql` in order, recording `schema_migrations`. `0001_init.sql` copied exactly from `docs/SPEC.md` section 2.
- [ ] `store` package behind an interface, so tests can use an in-memory SQLite database. No ORM.
- [ ] First-run setup: create the `settings` row with a random calendar token; generate VAPID keys into the data folder.
- [ ] Auth: argon2id passphrase check, session cookie, login rate limit, `X-Homebase` header check, `GET /api/me`, logout.
- [ ] `GET` and `POST /api/sync` per section 3, including cascade deletes, the 5-minute clock clamp, the `day_full` rule, and server-owned fields.
- [ ] Recurrence in the apply step (section 5).
- [ ] Daily rollover job, run at 00:05 and on the first sync of a new day.
- [ ] `POST /api/promote`.

**Tests required** (table-driven, in-memory DB): conflict resolution both directions; delete versus edit; cascade delete and detach; fourth task on a day; repeat spawn for `fixed` and `after_done`, including month ends; rollover increments `slipped` and clears the plan; promote moves attachments; idempotent re-push of the same ops.

**Done when:** a script that pushes ops from two simulated devices converges to the same state on both, and all tests pass.

## Phase 2: daily loop (web)

- [ ] Local data layer: IndexedDB mirror of all synced tables, an outbox, and the sync scheduler from section 3. Expose hooks such as `useTasks`, `useToday`. No component talks to `fetch` directly.
- [ ] App shell: phone bottom bar (Today, Inbox, Plan, Goals) and desktop left rail, as in `DESIGN.md`.
- [ ] Login screen.
- [ ] **Today:** goals as quiet lines, Your three (highlighter, rank, done state), capture bar, review link from Friday. Desktop: drag handle reorder. Phone: long-press drag plus up and down buttons in the task sheet.
- [ ] **Choose up to three** picker (phone) and the suggestions list (desktop): due soon, in progress, moved a few times.
- [ ] **Inbox:** sort with Today, Date, Goal, Project buttons (Project uses `/api/promote`).
- [ ] **Plan, Tasks:** week strip, filters (Everything, Due this week, Standalone, Repeating), grouped list, new-task form on desktop.
- [ ] Task sheet (phone) and panel (desktop): title, status, due, planned day, goal or project, repeat, notes.
- [ ] Service worker via `vite-plugin-pwa` (`injectManifest`), manifest, install prompt, offline shell.

**Done when:** with the network off you can capture, plan three tasks, complete one and reorder; turning the network on syncs everything to a second browser. Lighthouse PWA installability passes.

## Phase 3: goals, projects, attachments

- [ ] Goals list and goal detail (projects, tasks, notes, links).
- [ ] Plan, Projects: grouped by goal, add a project.
- [ ] Project detail: tasks, notes (saved on pause), attachments.
- [ ] Link and note attachments through sync.
- [ ] `POST /api/files`, `GET /api/files/{sha}`, upload UI (button and desktop drop area), orphan clean-up job, serving headers from section 6.
- [ ] Delete flows that ask: delete with contents or keep tasks standalone.

**Tests required:** upload size limit; same file twice stores once; hostile file served as download with `nosniff`; attachment owner check (exactly one owner).

**Done when:** a project with notes, a link, a note and two files survives a reload and shows on a second device.

## Phase 4: reminders and push

- [ ] `GET /api/push/key`, subscribe, unsubscribe, test.
- [ ] Web Push sender implemented to RFC 8291 and RFC 8292. Use the standard library plus `golang.org/x/crypto` only if needed. Include a unit test that encrypts a payload and decrypts it with the subscriber's key.
- [ ] Scheduler: 30-second ticker, `reminder_log` insert-first guard, 90-minute grace, message builders from section 6.
- [ ] Service worker `push` and `notificationclick` handlers; tapping opens Today and triggers a sync.
- [ ] Reminders screen (phone and desktop): three slots, toggles, time inputs, device list, test button, notification permission flow.

**Done when:** on an Android phone with the app installed and closed, the 08:00 notification arrives once and tapping it opens Today. Deploy first (HTTPS required) to test this.

## Phase 5: finish

- [ ] Recurrence UI (repeat picker: every N days, weeks, months; fixed or after done; end date).
- [ ] Weekly review (`/api/review`, `/api/review/complete`), phone and desktop.
- [ ] Calendar feed and rotate button.
- [ ] `homebase backup <dir>` command and restore note in the README.
- [ ] Deploy files in `deploy/`: Dockerfile, Caddyfile, systemd unit.
- [ ] Settings screen: time zone, week start, calendar link, sessions list.

**Done when:** a fresh install on a clean server follows the README top to bottom and works.

## Phase 6: polish (not designed yet)

These were never drawn. Design them in the same style as `DESIGN.md` and show the user before building.

- [ ] Dark theme (tokens are proposed in `DESIGN.md`; check contrast).
- [ ] Empty, loading and error states for every screen.
- [ ] Offline and sync-failed indicator.
- [ ] Phone Projects list, goal creation, and the repeat picker.
- [ ] Keyboard shortcuts on desktop (capture, switch screens).
- [ ] Accessibility pass: focus order, screen reader labels, reduced motion.

## Known risks

- Push needs HTTPS and an installed app on Android; test early on a real device.
- Row-level last-write-wins can drop one of two near-simultaneous edits of the same row. Accepted for one user.
- Google Calendar refreshes subscribed feeds only every several hours.
- The first Go build was never run in the sandbox this plan was written in (modules could not be downloaded), so expect small compile fixes when starting Phase 1.
