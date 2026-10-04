# Homebase

A calm personal planner for goals, projects and tasks, built to be opened every day rather than forgotten. It runs as an installable web app (PWA) on an Android phone and a computer, syncs between them, works offline, and sends up to three short reminders a day.

> **Status:** planned, not yet built. This repository currently contains the design and plan. Start with `PLAN.md`.

## What it does

- **Today** shows your goals as short lines and the three things you chose for today. Three is the limit on purpose.
- **Inbox** catches anything you type; you sort it later into a goal, a project or a day.
- **Plan** shows tasks by day with a week strip, and projects grouped under their goals.
- **Projects and goals** hold tasks, notes, links and files.
- **Weekly review** takes about a minute: keep, pause or drop what has gone quiet.
- **Reminders:** up to three a day (for example 08:00, 13:00, 20:00), each written from your current data.
- **Calendar feed:** an optional private link so Google Calendar can show due dates and planned tasks.
- **Repeating tasks** for chores, either on fixed days or after you finish.

## Documents

| File | Purpose |
| --- | --- |
| `AGENTS.md` | Rules and conventions for AI agents (and humans) working on the code |
| `PLAN.md` | Phased build plan with acceptance criteria |
| `DESIGN.md` | Visual design: tokens, type, components, screens |
| `docs/SPEC.md` | Database schema, sync protocol, API, behaviour rules |
| `mockups/` | Standalone HTML screens (phone and desktop); open `mockups/index.html` |

## Stack

- **Server:** Go, standard library HTTP, SQLite (`modernc.org/sqlite`, no CGO). One binary with the web app embedded.
- **Web:** React 19, TypeScript, React Router, Tailwind CSS v4, shadcn/ui, IndexedDB for the offline copy, a service worker for offline use and push.

## Quick start

Requires Go 1.26 or newer, Node 22.18 or newer, and `make`.

```sh
# once: store a passphrase hash for development in .env (never committed)
(cd server && go run ./cmd/homebase hash-passphrase) >> .env

# development: Go server and Vite dev server together, open http://localhost:5173
make dev

# tests
make test

# production build: web app embedded in a single binary at bin/homebase
make build
```

## Configuration

All configuration comes from environment variables.

| Variable | Default | Meaning |
| --- | --- | --- |
| `HOMEBASE_DATA` | `./data` | Folder for the database, uploaded files and push keys |
| `HOMEBASE_ADDR` | `:8080` | Listen address |
| `HOMEBASE_BASE_URL` | none | Public HTTPS address, used in the calendar link |
| `HOMEBASE_PASSPHRASE_HASH` | none, required | argon2id hash of your passphrase |
| `HOMEBASE_VAPID_SUBJECT` | `mailto:admin@localhost` | Contact sent to push services; use your real address |

Commands of the binary:

```sh
homebase hash-passphrase     # prompts for a passphrase and prints the hash
homebase serve               # runs the server
homebase backup <dir>        # copies the database (VACUUM INTO) and the files folder
```

## Deploying

Push notifications require HTTPS, so put the server behind one of these.

**Caddy on a server with a domain:**

```
homebase.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

**Tailscale (private, no public exposure):** run the server on the machine, then `tailscale serve --bg 8080`. Both your phone and computer need Tailscale running to open the app. The server itself needs outbound internet access to reach the push services.

Run the binary under systemd (`deploy/homebase.service`) or with Docker (`deploy/Dockerfile`).

## Installing on your devices

- **Android:** open the site in Chrome, choose the menu, then Install app. Allow notifications when asked. Reminders arrive even when the app is closed.
- **Computer:** in Chrome or Edge, use the install icon in the address bar.

## Backups

Run `homebase backup /path/to/backups` nightly from cron or a systemd timer. To restore, stop the server, replace the data folder with the backup, and start it again.

## Known limits

- Single user, one passphrase. There are no accounts.
- If two devices edit the same task within seconds, the later edit wins whole.
- Google Calendar refreshes subscribed feeds only every several hours, so changes appear slowly there.
- Reminder times follow one time zone set in Settings, not each device's.
