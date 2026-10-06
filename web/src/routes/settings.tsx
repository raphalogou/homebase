import { type FormEvent, useId, useMemo, useState } from "react";
import { Link, Navigate, NavLink, useParams } from "react-router";
import { PassphraseForm, UsernameForm } from "@/components/account-forms";
import { Columns } from "@/components/app-shell";
import { PassphraseField, TextField } from "@/components/form-field";
import {
  BackIcon,
  CheckIcon,
  ExternalIcon,
  LogOutIcon,
  NextIcon,
  NoticeIcon,
} from "@/components/icons";
import { Empty, ScreenTitle } from "@/components/section";
import { useShowShortcuts } from "@/components/shortcuts";
import { InlineError, SkeletonRows } from "@/components/states";
import { useNotify } from "@/components/toaster";
import { Button } from "@/components/ui/button";
import { DialogActions, ResponsiveDialog } from "@/components/ui/dialog";
import { Input, Label, Select } from "@/components/ui/input";
import { Segmented } from "@/components/ui/segmented";
import { ToggleSwitch } from "@/components/ui/toggle-switch";
import { ApiError, NetworkError } from "@/data/api";
import { ActionError, useAuth, useToday, useVersion, useWeekStart } from "@/data/hooks";
import { useData } from "@/data/provider";
import {
  useAccount,
  usePeople,
  useServerSettings,
  useSessions,
  useSettingsActions,
} from "@/data/settings";
import type { Person, SessionInfo } from "@/data/types";
import { USERNAME_HINT } from "@/lib/account";
import { changedOn } from "@/lib/dates";
import { setTheme, type Theme, useTheme } from "@/lib/theme";
import { labelFromUA } from "@/lib/ua";
import { useIsDesktop } from "@/lib/use-media";
import { cn } from "@/lib/utils";
import { COMMIT, REPOSITORY, VERSION } from "@/lib/version";

function ago(ms: number): string {
  const mins = Math.round((Date.now() - ms) / 60000);
  if (mins < 2) return "just now";
  if (mins < 60) return `${mins} min ago`;
  const hours = Math.round(mins / 60);
  if (hours < 36) return `${hours} h ago`;
  return `${Math.round(hours / 24)} days ago`;
}

const SECTIONS = [
  { value: "time", label: "Time and week" },
  { value: "calendar", label: "Calendar" },
  { value: "backups", label: "Backups" },
  { value: "account", label: "Account" },
  { value: "sessions", label: "Sessions" },
  { value: "people", label: "People" },
  { value: "about", label: "About" },
] as const;

type Section = (typeof SECTIONS)[number]["value"];

// Approved design (DESIGN.md "Settings"): a screen like the others. On
// desktop the sections are a nav at the left of the content
// (/settings/account); on the phone /settings lists them and each opens on
// its own, with a link back. Changing the username or passphrase and
// removing a person open a dialog, a sheet on the phone.
export default function Settings() {
  const { section: param } = useParams();
  const desktop = useIsDesktop();
  const showShortcuts = useShowShortcuts();
  const notify = useNotify();
  const { store } = useData();
  useVersion();
  const owner = store.me?.owner === true;
  // People is the owner's alone.
  const sections = SECTIONS.filter((s) => owner || s.value !== "people");
  const section = sections.find((s) => s.value === param);

  async function run(fn: () => Promise<unknown>, done?: string) {
    try {
      await fn();
      if (done) notify({ title: done, icon: CheckIcon });
    } catch (err) {
      if (err instanceof ActionError) notify({ title: err.message, icon: NoticeIcon });
      else throw err;
    }
  }

  if (param && !section) return <Navigate to="/settings" replace />;
  if (!section) {
    if (desktop) return <Navigate to="/settings/time" replace />;
    return (
      <Columns
        main={
          <>
            <ScreenTitle>Settings</ScreenTitle>
            <nav aria-label="Settings" className="mt-6">
              <ul>
                {sections.map((s) => (
                  <li key={s.value} className="border-t border-line">
                    <Link
                      to={`/settings/${s.value}`}
                      className="flex min-h-14 items-center justify-between text-[17px]"
                    >
                      {s.label}
                      <NextIcon size={20} className="text-muted-foreground" />
                    </Link>
                  </li>
                ))}
              </ul>
            </nav>
          </>
        }
      />
    );
  }

  const body = <SectionBody section={section.value} run={run} />;
  if (!desktop) {
    return (
      <Columns
        main={
          <>
            <Link
              to="/settings"
              className="-ml-2.5 inline-flex min-h-11 items-center gap-0.5 pr-2 font-semibold"
            >
              <BackIcon size={24} />
              Settings
            </Link>
            <ScreenTitle className="mt-2">{section.label}</ScreenTitle>
            <div className="mt-6">{body}</div>
          </>
        }
      />
    );
  }
  return (
    <Columns
      main={
        <>
          <ScreenTitle>Settings</ScreenTitle>
          <div className="mt-10 flex gap-12">
            <nav aria-label="Settings" className="flex w-44 shrink-0 flex-col gap-1">
              {sections.map((s) => (
                <NavLink
                  key={s.value}
                  to={`/settings/${s.value}`}
                  className={({ isActive }) =>
                    cn(
                      "flex min-h-11 items-center rounded-md px-3 text-[15px] font-medium hover:bg-soft",
                      isActive && "bg-soft font-semibold",
                    )
                  }
                >
                  {s.label}
                </NavLink>
              ))}
              <Button variant="text" className="mt-6 self-start" onClick={showShortcuts}>
                Keyboard shortcuts
              </Button>
            </nav>
            <section aria-labelledby="settings-section" className="min-w-0 flex-1">
              <h2 id="settings-section" className="mb-4 text-[22px]/[1.2] font-bold">
                {section.label}
              </h2>
              {body}
            </section>
          </div>
        </>
      }
    />
  );
}

function SectionBody({ section, run }: { section: Section; run: Run }) {
  const { logout } = useAuth();
  switch (section) {
    case "time":
      return <TimeAndWeek />;
    case "calendar":
      return <Calendar run={run} />;
    case "backups":
      return <Backups run={run} />;
    case "account":
      return <Account />;
    case "sessions":
      return (
        <>
          <Sessions run={run} />
          <div className="mt-8 border-t border-line pt-6">
            <Button variant="secondary" onClick={() => void logout()}>
              <LogOutIcon size={20} />
              Log out of this device
            </Button>
          </div>
        </>
      );
    case "people":
      return <People />;
    case "about":
      return <About />;
  }
}

type Run = (fn: () => Promise<unknown>, done?: string) => Promise<void>;

function TimeAndWeek() {
  const { store } = useData();
  const weekStart = useWeekStart();
  const actions = useSettingsActions();
  const id = useId();
  const theme = useTheme();
  const notify = useNotify();
  // A failed save shows under the control it belongs to.
  const [failed, setFailed] = useState<"tz" | "week" | null>(null);
  async function save(which: "tz" | "week", zone: string, week: 0 | 1, done?: string) {
    setFailed(null);
    try {
      await actions.saveTime(zone, week);
      if (done) notify({ title: done, icon: CheckIcon });
    } catch (err) {
      if (!(err instanceof ActionError)) throw err;
      setFailed(which);
    }
  }
  const tz = store.me?.tz ?? "UTC";
  const zones = useMemo(() => {
    const all = Intl.supportedValuesOf("timeZone");
    return all.includes(tz) ? all : [tz, ...all];
  }, [tz]);

  return (
    <div>
      <div className="flex flex-col gap-4">
        <div>
          <Label htmlFor={`${id}-tz`}>Time zone</Label>
          <Select
            id={`${id}-tz`}
            value={tz}
            onChange={(e) => void save("tz", e.target.value, weekStart, "Time zone saved")}
          >
            {zones.map((z) => (
              <option key={z} value={z}>
                {z.replace(/_/g, " ")}
              </option>
            ))}
          </Select>
          <p className="mt-1.5 text-[13px] text-muted-foreground">
            Days turn over and reminders go out in this zone, on every device.
          </p>
          {failed === "tz" && <InlineError>Could not save. Try again.</InlineError>}
        </div>
        <div>
          <span className="mb-2 block text-sm font-semibold text-muted-foreground">
            Week starts on
          </span>
          <Segmented<"1" | "0">
            label="Week starts on"
            className="max-w-xs"
            value={weekStart === 1 ? "1" : "0"}
            options={[
              { value: "1", label: "Monday" },
              { value: "0", label: "Sunday" },
            ]}
            onChange={(v) => void save("week", tz, v === "1" ? 1 : 0)}
          />
          {failed === "week" && <InlineError>Could not save. Try again.</InlineError>}
        </div>
        <div>
          <span className="mb-2 block text-sm font-semibold text-muted-foreground">Theme</span>
          <Segmented<Theme>
            label="Theme"
            className="max-w-sm"
            value={theme}
            options={[
              { value: "system", label: "System" },
              { value: "light", label: "Light" },
              { value: "dark", label: "Dark" },
            ]}
            onChange={setTheme}
          />
          <p className="mt-1.5 text-[13px] text-muted-foreground">On this device only.</p>
        </div>
      </div>
    </div>
  );
}

function Calendar({ run }: { run: Run }) {
  const [settings] = useServerSettings();
  const actions = useSettingsActions();
  const [url, setUrl] = useState<string | null>(null);
  const id = useId();
  const shown = url ?? (settings.state === "ready" ? settings.data.calendarUrl : null);

  return (
    <div>
      <div>
        <p className="text-muted-foreground">
          Add this private link to Google Calendar ("From URL") to see due and planned tasks there.
          Google refreshes it every few hours, so changes show up slowly.
        </p>
        {settings.state === "loading" && !url && (
          <div className="mt-4">
            <SkeletonRows kind="line" count={1} label="Loading the calendar link" />
          </div>
        )}
        {settings.state === "offline" && <Empty>The calendar link needs a connection.</Empty>}
        {shown && (
          <>
            <Label htmlFor={`${id}-url`} className="mt-4">
              Private link
            </Label>
            <div className="flex gap-2">
              <Input
                id={`${id}-url`}
                readOnly
                value={shown}
                onFocus={(e) => e.currentTarget.select()}
              />
              <Button
                className="h-12 shrink-0 px-4 text-sm"
                onClick={() => void run(() => navigator.clipboard.writeText(shown), "Link copied")}
              >
                Copy
              </Button>
            </div>
            <p className="mt-1.5 text-[13px] text-muted-foreground">
              Anyone with the link can read your task titles.
            </p>
            <Button
              variant="text"
              className="mt-2"
              onClick={() =>
                void run(
                  async () => setUrl(await actions.rotateCalendar()),
                  "New link made. The old one stopped working.",
                )
              }
            >
              Make a new link
            </Button>
          </>
        )}
      </div>
    </div>
  );
}

/** "day", "week", "3 days", "6 hours": the end of "Back up every …". */
function every(hours: number): string {
  if (hours === 24) return "day";
  if (hours === 168) return "week";
  if (hours % 24 === 0) return `${hours / 24} days`;
  return hours === 1 ? "hour" : `${hours} hours`;
}

function Backups({ run }: { run: Run }) {
  const [settings] = useServerSettings();
  const { store } = useData();
  const today = useToday();
  const actions = useSettingsActions();
  const [on, setOn] = useState<boolean | null>(null);
  if (settings.state === "loading") {
    return <SkeletonRows kind="line" count={2} label="Loading backups" />;
  }
  if (settings.state !== "ready") return <Empty>Backups need a connection.</Empty>;
  const { backupDir, backupHours, lastBackupAt } = settings.data;
  const label = `Back up every ${every(backupHours)}`;
  const enabled = on ?? settings.data.backups;
  const tz = store.me?.tz ?? "UTC";

  return (
    <div>
      <div className="flex min-h-16 items-center gap-4">
        <div className="min-w-0 flex-1">
          <p className="text-[17px] font-medium">{label}</p>
          <p className="text-[13px] text-muted-foreground">
            The server copies your tasks, files and reminder key.
          </p>
        </div>
        <ToggleSwitch
          label={label}
          checked={enabled}
          onChange={(v) =>
            void run(
              async () => {
                await actions.saveBackups(v);
                setOn(v);
              },
              v ? "Backups on" : "Backups off",
            )
          }
        />
      </div>
      <dl className="mt-4 border-t border-line pt-4 text-[15px]">
        <dt className="text-[13px] text-muted-foreground">Folder on the server</dt>
        <dd className="mt-0.5 break-all font-mono text-sm">{backupDir}</dd>
        <dt className="mt-4 text-[13px] text-muted-foreground">Last backup</dt>
        <dd className="mt-0.5">
          {lastBackupAt
            ? `${changedOn(lastBackupAt, tz, today)}, ${new Date(lastBackupAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", timeZone: tz })}`
            : enabled
              ? "None yet. The first is made within a minute."
              : "None yet"}
        </dd>
      </dl>
      <p className="mt-4 text-[13px] text-muted-foreground">
        A copy on the same disk does not survive losing that disk. Set HOMEBASE_BACKUP_DIR to
        another one, or copy this folder elsewhere now and then.
      </p>
    </div>
  );
}

function People() {
  const [people, reload] = usePeople();
  const actions = useSettingsActions();
  const notify = useNotify();
  // The dialog keeps showing who it was opened for while it closes.
  const [removing, setRemoving] = useState<Person | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [username, setUsername] = useState("");
  const [passphrase, setPassphrase] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<{ field: "username" | "passphrase"; message: string } | null>(
    null,
  );

  async function add(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      const p = await actions.addPerson(username, passphrase);
      notify({ title: `Added ${p.username}`, icon: CheckIcon });
      setUsername("");
      setPassphrase("");
      reload();
    } catch (err) {
      if (err instanceof ApiError)
        setError({
          field: err.field === "username" ? "username" : "passphrase",
          message: err.message,
        });
      else if (err instanceof NetworkError)
        setError({ field: "passphrase", message: "Adding a person needs a connection." });
      else throw err;
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <p className="text-muted-foreground">
        Everyone has their own planner on this server: their own tasks, reminders and settings.
        Nobody sees anyone else's.
      </p>
      {people.state === "loading" && (
        <div className="mt-4">
          <SkeletonRows kind="line" count={2} label="Loading people" />
        </div>
      )}
      {people.state === "offline" && <Empty>The list of people needs a connection.</Empty>}
      {people.state === "ready" && (
        <ul className="mt-4">
          {people.data.map((p) => (
            <li key={p.id} className="flex min-h-16 items-center gap-3 border-t border-line py-2.5">
              <div className="min-w-0 flex-1">
                <p className="truncate text-[17px]">{p.username}</p>
                <p className="text-[13px] text-muted-foreground">
                  {p.owner ? "Owner" : p.mustChange ? "Has not chosen a passphrase yet" : "Member"}
                </p>
              </div>
              {!p.owner && (
                <Button
                  variant="text"
                  aria-label={`Remove ${p.username}`}
                  onClick={() => {
                    setRemoving(p);
                    setConfirming(true);
                  }}
                >
                  Remove
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}
      <ResponsiveDialog
        open={confirming}
        onOpenChange={setConfirming}
        title={removing ? `Remove ${removing.username}?` : "Remove"}
      >
        {removing && (
          <RemovePerson
            person={removing}
            onDone={(removed) => {
              setConfirming(false);
              if (removed) reload();
            }}
          />
        )}
      </ResponsiveDialog>
      <form
        onSubmit={add}
        noValidate
        className="mt-8 flex flex-col gap-4 border-t border-line pt-6"
      >
        <h3 className="font-semibold">Add a person</h3>
        <TextField
          label="Username"
          placeholder={USERNAME_HINT}
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
          value={username}
          onChange={(e) => {
            setUsername(e.target.value);
            setError(null);
          }}
          error={error?.field === "username" ? error.message : undefined}
        />
        <PassphraseField
          label="One-time passphrase"
          hint="Tell them yourself. They choose their own when they first log in."
          autoComplete="new-password"
          value={passphrase}
          onChange={(e) => {
            setPassphrase(e.target.value);
            setError(null);
          }}
          error={error?.field === "passphrase" ? error.message : undefined}
        />
        <Button type="submit" className="self-start" disabled={busy}>
          {busy ? "Adding" : "Add person"}
        </Button>
      </form>
    </div>
  );
}

/** Confirms removing a person, inside the dialog People opens. */
function RemovePerson({ person, onDone }: { person: Person; onDone: (removed: boolean) => void }) {
  const actions = useSettingsActions();
  const notify = useNotify();
  const [busy, setBusy] = useState(false);

  async function remove() {
    setBusy(true);
    try {
      await actions.removePerson(person.id);
      notify({ title: `Removed ${person.username}`, icon: CheckIcon });
      onDone(true);
    } catch (err) {
      setBusy(false);
      if (err instanceof ActionError) notify({ title: err.message, icon: NoticeIcon });
      else throw err;
    }
  }

  return (
    <>
      <p className="text-muted-foreground">
        They can no longer log in, and their planner moves to the server's removed folder. Moving it
        back undoes this.
      </p>
      <DialogActions>
        <Button onClick={() => void remove()} disabled={busy}>
          {busy ? "Removing" : `Remove ${person.username}`}
        </Button>
        {/* Focus starts on Cancel, so a stray Enter does nothing drastic. */}
        <Button variant="text" className="self-center" autoFocus onClick={() => onDone(false)}>
          Cancel
        </Button>
      </DialogActions>
    </>
  );
}

function About() {
  const build = COMMIT === "unknown" ? null : `${REPOSITORY}/commit/${COMMIT}`;
  const link = (href: string, label: string, detail: string) => (
    <li className="border-t border-line first:border-t-0">
      <a
        href={href}
        target="_blank"
        rel="noreferrer"
        className="-mx-3 flex min-h-16 items-center gap-3 rounded-md px-3 py-2.5 hover:bg-soft focus-visible:outline-offset-[-2px]"
      >
        <span className="min-w-0 flex-1">
          <span className="block font-medium">{label}</span>
          <span className="block truncate text-[13px] text-muted-foreground">{detail}</span>
        </span>
        <ExternalIcon size={18} className="shrink-0 text-muted-foreground" />
        <span className="sr-only">(opens in a new tab)</span>
      </a>
    </li>
  );

  return (
    <div>
      <div className="flex items-center gap-4">
        <img src="/icon.svg" alt="" className="size-14 rounded-[13px] border border-line" />
        <div>
          <p className="text-[22px]/[1.2] font-bold">Homebase</p>
          <p className="mt-0.5 font-mono text-[13px] text-muted-foreground">
            {VERSION}
            {build && (
              <>
                {" · "}
                <a
                  href={build}
                  target="_blank"
                  rel="noreferrer"
                  className="underline-offset-2 hover:underline"
                >
                  {COMMIT}
                </a>
              </>
            )}
          </p>
        </div>
      </div>
      <p className="mt-5 text-muted-foreground">
        A calm planner for one person. Goals give your tasks a reason, each day you choose up to
        three, and up to three reminders a day keep it in mind. It works offline and syncs when you
        are back.
      </p>
      <ul className="mt-6">
        {link(REPOSITORY, "Source code", REPOSITORY.replace("https://", ""))}
        {link(`${REPOSITORY}/issues`, "Report a problem", "Open an issue on GitHub")}
      </ul>
    </div>
  );
}

const CHANGES = {
  username: {
    label: "Username",
    title: "Change username",
    description: "You will use the new username to log in on every device.",
    Form: UsernameForm,
    saved: "Username saved",
  },
  passphrase: {
    label: "Passphrase",
    title: "Change passphrase",
    description: "Other devices will be logged out. This one stays logged in.",
    Form: PassphraseForm,
    saved: "Passphrase saved. Other devices were logged out.",
  },
} as const;

type Change = keyof typeof CHANGES;

function Account() {
  const { store } = useData();
  const today = useToday();
  const [account, reload] = useAccount();
  const notify = useNotify();
  // The dialog keeps showing what it was opened for while it closes.
  const [what, setWhat] = useState<Change>("username");
  const [open, setOpen] = useState(false);
  const info = account.state === "ready" ? account.data : null;
  const values: Record<Change, string> = {
    username: info?.username ?? store.me?.username ?? "Not loaded",
    passphrase: info
      ? `Changed ${changedOn(info.passphraseChangedAt, store.me?.tz ?? "UTC", today)}`
      : "Set",
  };
  const change = CHANGES[what];

  return (
    <div>
      <ul>
        {(Object.keys(CHANGES) as Change[]).map((key) => (
          <li
            key={key}
            className="flex min-h-16 items-center gap-3 border-t border-line py-2.5 first:border-t-0"
          >
            <div className="min-w-0 flex-1">
              <p className="text-[13px] text-muted-foreground">{CHANGES[key].label}</p>
              <p className="truncate text-[17px]">{values[key]}</p>
            </div>
            <Button
              variant="text"
              aria-label={`Change ${key}`}
              disabled={account.state === "offline"}
              onClick={() => {
                setWhat(key);
                setOpen(true);
              }}
            >
              Change
            </Button>
          </li>
        ))}
      </ul>
      {account.state === "offline" && (
        <p className="mt-1 text-[13px] text-muted-foreground">Changing these needs a connection.</p>
      )}
      <ResponsiveDialog
        open={open}
        onOpenChange={setOpen}
        title={change.title}
        description={change.description}
      >
        <change.Form
          key={what}
          onClose={() => setOpen(false)}
          onSaved={() => {
            notify({ title: change.saved, icon: CheckIcon });
            reload();
          }}
        />
      </ResponsiveDialog>
    </div>
  );
}

function Sessions({ run }: { run: Run }) {
  const [sessions, reload] = useSessions();
  const actions = useSettingsActions();
  return (
    <div>
      {sessions.state === "loading" && (
        <SkeletonRows kind="line" count={2} label="Loading your sessions" />
      )}
      {sessions.state === "offline" && <Empty>The list of sessions needs a connection.</Empty>}
      {sessions.state === "ready" && (
        <ul>
          {sessions.data.map((s: SessionInfo) => (
            <li
              key={s.id}
              className="flex min-h-16 items-center gap-3 border-t border-line py-2.5 first:border-t-0"
            >
              <div className="min-w-0 flex-1">
                <p className="font-medium">{labelFromUA(s.label) || "Unknown browser"}</p>
                <p className="text-[13px] text-muted-foreground">
                  {s.current ? "This device" : `Last used ${ago(s.lastSeen)}`}
                </p>
              </div>
              {!s.current && (
                <Button
                  variant="text"
                  aria-label={`Log out ${labelFromUA(s.label) || "this session"}`}
                  onClick={() =>
                    void run(async () => {
                      await actions.revoke(s.id);
                      reload();
                    }, "Session logged out")
                  }
                >
                  Log out
                </Button>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
