# Homebase

A calm personal planner for goals, projects and tasks, built to be opened every day rather than forgotten. It runs as an installable web app (PWA) on an Android phone and a computer, syncs between them, works offline, and sends up to three short reminders a day.

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

## Stack

- **Server:** Go, standard library HTTP, SQLite (`modernc.org/sqlite`, no CGO). One binary with the web app embedded.
- **Web:** React 19, TypeScript, React Router, Tailwind CSS v4, shadcn/ui on Base UI, IndexedDB for the offline copy, a service worker for offline use and push.

## Development

Requires Go 1.26 or newer, Node 22.18 or newer, and `make`.

```sh
# Go server and Vite dev server together; open http://localhost:5173
# The first visit shows Set up, where you choose a username and passphrase.
make dev

# lint, vet, Go tests, web type check and unit tests
make test

# production build: the web app embedded in one binary at bin/homebase
make build
```

## Install on a server

You need a server with a domain name pointing at it (or Tailscale, see below). Reminders use Web Push, which only works over HTTPS. Pick one of the two ways to run Homebase, then set up HTTPS and backups.

### Option A: Docker

1. **Build the image** from a clone of this repository:

   ```sh
   docker build -f deploy/Dockerfile -t homebase \
     --build-arg VERSION=$(git describe --tags --always) --build-arg COMMIT=$(git rev-parse --short HEAD) .
   ```

   Without the two `--build-arg`s it still builds, but Settings, About shows "dev".

   Or pull the one CI publishes from `main` (`.github/workflows/publish.yml`; a `v1.2.0` tag also publishes `:1.2.0`) and name it `homebase` for the steps below:

   ```sh
   docker pull ghcr.io/raphalogou/homebase:latest
   docker tag ghcr.io/raphalogou/homebase:latest homebase
   ```

2. **Write the settings file:**

   ```sh
   echo 'HOMEBASE_BASE_URL=https://homebase.example.com' > homebase.env
   echo 'HOMEBASE_VAPID_SUBJECT=mailto:you@example.com' >> homebase.env
   chmod 600 homebase.env
   ```

3. **Start it.** Data lives in the `homebase-data` volume; the port is only open to the machine itself, for the proxy:

   ```sh
   docker run -d --name homebase --restart unless-stopped \
     --env-file homebase.env \
     -v homebase-data:/data \
     -p 127.0.0.1:8080:8080 \
     homebase
   curl http://127.0.0.1:8080/healthz   # prints: ok
   docker ps                            # STATUS shows "healthy" after a minute
   ```

### Option B: binary and systemd

1. **Build** on any machine with Go and Node, then copy the binary to the server:

   ```sh
   make build
   scp bin/homebase server:/tmp/homebase
   ```

2. **On the server**, install it, add a user, and write the settings file:

   ```sh
   sudo install -m 755 /tmp/homebase /usr/local/bin/homebase
   sudo useradd --system --home /var/lib/homebase --shell /usr/sbin/nologin homebase
   sudo install -d -m 700 /etc/homebase
   echo 'HOMEBASE_BASE_URL=https://homebase.example.com' | sudo tee /etc/homebase/env > /dev/null
   echo 'HOMEBASE_VAPID_SUBJECT=mailto:you@example.com' | sudo tee -a /etc/homebase/env > /dev/null
   sudo chmod 600 /etc/homebase/env
   ```

3. **Start it** with the unit in `deploy/`. systemd creates `/var/lib/homebase` for the data:

   ```sh
   sudo cp deploy/homebase.service /etc/systemd/system/
   sudo systemctl daemon-reload
   sudo systemctl enable --now homebase
   curl http://127.0.0.1:8080/healthz   # prints: ok
   ```

### HTTPS

**Caddy** (a public domain): install Caddy, put `deploy/Caddyfile` in `/etc/caddy/Caddyfile` with your domain instead of `homebase.example.com`, and run `sudo systemctl reload caddy`. Caddy gets the certificate itself.

**Tailscale** (private, no public exposure): run `tailscale serve --bg 8080` on the server and use the address it prints as `HOMEBASE_BASE_URL`. Your phone and computer need Tailscale running to reach Homebase. The server still needs outbound internet access for the push services.

Then open the address in a browser straight away. The first visit shows **Set up Homebase**, where you choose a username and a passphrase of at least 12 characters; you are then logged in. Until that is done, anyone who reaches the address could do it instead, so do it as soon as the server is up. Both can be changed later in Settings.

To create the owner's account without the browser, for example on a server that is public from the start, put the line from `homebase hash-passphrase` in the settings file before the first start. The account is then created with the username `owner` and that passphrase. An install from before usernames gets the same: log in as `owner` with your passphrase and rename it in Settings. After the account exists the variable is no longer read.

### Versions

The version comes from git: a tag `v1.2.0` gives `v1.2.0`, later commits `v1.2.0-3-gabc1234`. `homebase version` prints it with the commit, the server logs it at start, and Settings, About shows both. To release, tag and push. CI publishes the image as `:1.2.0` for amd64 and arm64, then creates the GitHub release: the `docker pull` line, the commits since the previous tag, and GitHub's generated notes:

```sh
git tag v1.2.0 && git push origin v1.2.0
```

### Backups

`homebase backup <dir>` copies the database (safely, while the server runs), the uploaded files and the push key into `<dir>`. Each run adds a dated copy of the database; files are shared between runs, so only new ones are copied. Old database copies are not removed; delete them as you see fit.

**From Settings (any install):** turn on "Back up every day" in Settings, Backups. The server then makes a backup when the newest is older than `HOMEBASE_BACKUP_INTERVAL` (one day by default) or missing, checked each minute, so the first comes right away and one missed while it was off follows a restart. They go to `backups` in the data folder, which needs no setup but sits on the same disk. To keep them on another disk, set `HOMEBASE_BACKUP_DIR`; with Docker, mount a folder there:

```sh
sudo install -d -o 65532 -g 65532 -m 700 /var/backups/homebase
echo 'HOMEBASE_BACKUP_DIR=/backups' >> homebase.env
docker rm -f homebase
docker run -d --name homebase --restart unless-stopped --env-file homebase.env \
  -v homebase-data:/data -v /var/backups/homebase:/backups -p 127.0.0.1:8080:8080 homebase
```

A backup on demand: `docker exec homebase homebase backup /backups`, or `homebase backup <dir>`.

**systemd timer (instead of the switch):** the timer in `deploy/` runs it at 03:30:

```sh
sudo install -d -o homebase -g homebase -m 700 /var/backups/homebase
sudo cp deploy/homebase-backup.service deploy/homebase-backup.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now homebase-backup.timer
```

### Restore

Each person's planner is a folder `users/<id>/` in the data folder (`/var/lib/homebase`, or the `homebase-data` volume), and their backups are in `<backup folder>/<id>/`. Each person's Settings, Backups shows their folder, which ends in their id. To restore one person:

1. Stop Homebase (`docker stop homebase` or `sudo systemctl stop homebase`).
2. In `users/<id>/`, delete `homebase.db` and any `homebase.db-wal` and `homebase.db-shm`.
3. Copy the backup you want in as `homebase.db`, and copy the backup's `files` folder over the one there. Keep the owner as the Homebase user (`homebase`, or `65532` for Docker). `vapid-private.pem` is shared by everyone and sits at the top of the data folder and of a full `homebase backup`.
4. Start Homebase again. Devices sync what changed; anything newer on a device than the backup is sent up again.

A removed person's folder is moved to `removed/<id>-<time>/` in the data folder. To bring them back, stop Homebase, move it back to `users/<id>/`, and start it.

## Configuration

All configuration comes from environment variables.

| Variable | Default | Meaning |
| --- | --- | --- |
| `HOMEBASE_PASSPHRASE_HASH` | none | Optional. Creates the account (username `owner`) on first start, from `homebase hash-passphrase`; ignored once the account exists |
| `HOMEBASE_DATA` | `./data` | Folder for the database, uploaded files and the push key |
| `HOMEBASE_ADDR` | `:8080` | Listen address |
| `HOMEBASE_BASE_URL` | the address requests come to | Public HTTPS address, used in the calendar link |
| `HOMEBASE_VAPID_SUBJECT` | `mailto:admin@localhost` | Contact sent to push services; use your real address |
| `HOMEBASE_BACKUP_DIR` | `backups` in the data folder | Where the backups go once switched on in Settings |
| `HOMEBASE_BACKUP_INTERVAL` | `1d` | How often: whole days (`7d` for a week) or a Go duration (`12h`), at least `1h` |

Commands of the binary:

```sh
homebase serve               # runs the server
homebase hash-passphrase     # prompts for a passphrase and prints the settings line
homebase backup <dir>        # copies the database, files and push key into <dir>
```

## Installing on your devices

- **Android:** open the address in Chrome, open the menu, then choose Install app. Then open Reminders (at the foot of Today), tap "Get reminders on this device" and allow notifications. Reminders arrive even when the app is closed. "Send a test now" checks it works.
- **Computer:** in Chrome or Edge, use the install icon in the address bar.
- **Calendar:** in Settings, copy the private calendar link and add it in Google Calendar under "Other calendars", "From URL".

## Known limits

- Several people can share a server, each with a private planner; the owner adds and removes them in Settings, People. Nothing is shared between planners.
- If two devices edit the same task within seconds, the later edit wins whole.
- Uploading files needs a connection; everything else works offline.
- Google Calendar refreshes subscribed feeds only every several hours, so changes appear slowly there.
- Reminder times follow one time zone set in Settings, not each device's.
