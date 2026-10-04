# AGENTS.md

Instructions for AI coding agents working on Homebase. Read this file first, then `PLAN.md`, `docs/SPEC.md` and `DESIGN.md`. Open `mockups/index.html` to see the screens.

## What this is

Homebase is a personal planner for one person, used on an Android phone and a computer. It holds goals, projects and tasks, shows a calm daily view, and sends up to three reminders a day. The aim is that the user does not forget it exists, so every decision favours a short daily ritual over a feature-rich task manager.

## Principles (these decide close calls)

1. **Calm, never nagging.** No red alerts, no streaks, no guilt. A missed plan quietly returns to the pile.
2. **Three is the limit.** At most 3 tasks planned per day. Never raise this without being asked.
3. **The user chooses.** Suggestions are fine; auto-picking is not.
4. **Every task shows its why.** Tasks link to a project or goal when they have one.
5. **Fast and offline.** Every action works without a network and syncs later.
6. **Small.** Prefer deleting code to adding options. Do not add features that are not in `PLAN.md`.

## Stack (fixed; do not swap)

- **Server:** Go 1.26 or newer (required by `modernc.org/sqlite`), standard library `net/http` with method patterns, `modernc.org/sqlite`, `log/slog`. Extra modules only when unavoidable (`golang.org/x/crypto` for argon2id, optionally a Web Push library). No web framework, no ORM.
- **Web:** React 19, TypeScript with `strict`, React Router (library mode), Tailwind CSS v4, shadcn/ui components, `idb` for IndexedDB, `vite-plugin-pwa` in `injectManifest` mode. Fonts through `@fontsource`.
- **Tooling:** Vite, Biome for web lint and formatting, the Node built-in test runner (`node --test`) for web unit tests, `gofmt` and `go vet` for Go.
- **Deploy:** one Go binary that embeds the built web app, one data folder, HTTPS in front (Caddy or Tailscale).

## Layout

```
README.md  AGENTS.md  DESIGN.md  PLAN.md
docs/SPEC.md                  schema, sync, API (source of truth)
mockups/                      static HTML screens, visual reference only
server/
  cmd/homebase/main.go
  internal/{config,db,migrate,store,syncer,recur,api,apperr,auth,push,sched,review,ics,files,webui}/
  migrations/0001_init.sql   embedded by migrations/migrations.go
web/
  src/{routes,components,components/ui,data,lib,styles}/
  src/sw.ts                   service worker (push, notificationclick, precache)
  public/                     manifest icons
deploy/                       Dockerfile, Caddyfile, homebase.service
```

## Commands

Create a `Makefile` in Phase 0 with these targets and keep them working:

- `make dev`: Go server and Vite dev server, API proxied.
- `make build`: build the web app, then the Go binary with the web app embedded.
- `make test`: `go test ./...`, `go vet ./...`, web type check, lint, unit tests.
- `make lint`: `gofmt` check and Biome.
- `make fmt`: rewrite Go and web files in place.
- `make ci`: clean dependency install, then `make test` and `make build`.

Run `make test` before you say a task is done.

## Rules

### Data and sync

- The schema in `docs/SPEC.md` is fixed. Migrations are append-only: add `0002_*.sql`, never edit an applied file.
- Every write to a synced table must increment `meta.rev` and stamp `rev` and `updated_at` in the same transaction.
- Dates such as `due` and `planned_on` are local-day `YYYY-MM-DD` strings. Never convert them through `Date` objects in a way that shifts the day. Timestamps are UTC milliseconds.
- Business rules (3 per day, rollover, recurrence, cascade delete) live on the server. The client may anticipate them for speed but the server decides.
- Components never call `fetch`. They use hooks from `web/src/data/`.
- Store user data in IndexedDB, not `localStorage`.

### Code

- Keep SQL in `internal/store` behind an interface so tests can use an in-memory database.
- Table-driven tests for anything in `syncer`, `recur`, `sched` and `review`.
- Handle every error. Return typed errors that map to the codes in `docs/SPEC.md`.
- No global mutable state. Pass dependencies explicitly. Configuration comes from environment variables only.
- Comments explain why, not what. No commented-out code.
- Web: functional components, no class components, no `any`, no default exports except route modules.

### UI

- Follow `DESIGN.md` exactly: tokens, type, spacing, the highlighter mark, the way lists and rows are built.
- Use shadcn/ui primitives and restyle them with the tokens. Do not add another component library.
- Do not add gradients, drop shadows (except the lifted drag row), emoji, card grids or left-border cards.
- Touch targets are at least 44 px. Every icon-only button has an `aria-label`. Every input has a label.
- Copy is sentence case, plain verbs, no exclamation marks, no filler. Buttons say what happens ("Add task", "Finish review").

### Security

- Never log passphrases, tokens or push keys. Never commit `.env`, the data folder or VAPID keys.
- Compare secrets in constant time. Store only the SHA-256 of session tokens.
- Serve uploaded files with `nosniff`, and as downloads unless image or PDF with the sandbox CSP.
- Validate all input lengths and enums at the API boundary even though the database also checks them.
- No third-party scripts or fonts loaded from the network.

## Ask the user before

- Adding a dependency not named above.
- Changing the schema, the sync protocol or an API shape in `docs/SPEC.md` (propose the change and update the spec in the same commit).
- Changing anything in the Principles section.
- Designing a screen that is not in `DESIGN.md`. Phase 6 items must be shown to the user first.
- Anything that sends data to a third party.

## Definition of done

A task is done when: the behaviour works offline and online; tests cover the rules it touches; `make test` passes; the UI matches `DESIGN.md` at phone width (390 px) and desktop width (1280 px); keyboard and screen reader basics work; `PLAN.md` boxes are ticked; and any change to behaviour is reflected in `docs/SPEC.md`.

## Pitfalls seen in planning

- Web Push works only over HTTPS, and on Android only reliably when the app is installed. Test on a real phone.
- Google Calendar polls subscribed feeds every several hours. Do not promise instant calendar updates in the UI.
- Day boundaries use `settings.tz`, not the server's zone and not the browser's at request time.
- A done task still occupies one of the day's three slots.
