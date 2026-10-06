import { Tabs } from "@base-ui/react/tabs";
import { type FormEvent, useId, useMemo, useState } from "react";
import { ACCOUNT_VIEWS } from "@/components/account-forms";
import { PassphraseField, TextField } from "@/components/form-field";
import { CheckIcon, ExternalIcon, LogOutIcon, NoticeIcon } from "@/components/icons";
import { Empty } from "@/components/section";
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
import type { SessionInfo } from "@/data/types";
import { USERNAME_HINT } from "@/lib/account";
import { changedOn } from "@/lib/dates";
import { useModal } from "@/lib/modal";
import { setTheme, type Theme, useTheme } from "@/lib/theme";
import { labelFromUA } from "@/lib/ua";
import { useIsDesktop } from "@/lib/use-media";
import { COMMIT, REPOSITORY, VERSION } from "@/lib/version";

function ago(ms: number): string {
  const mins = Math.round((Date.now() - ms) / 60000);
  if (mins < 2) return "just now";
  if (mins < 60) return `${mins} min ago`;
  const hours = Math.round(mins / 60);
  if (hours < 36) return `${hours} h ago`;
  return `${Math.round(hours / 24)} days ago`;
}

const TABS = [
  { value: "time", label: "Time and week" },
  { value: "calendar", label: "Calendar" },
  { value: "backups", label: "Backups" },
  { value: "account", label: "Account" },
  { value: "sessions", label: "Sessions" },
  { value: "people", label: "People" },
  { value: "about", label: "About" },
] as const;

type Tab = (typeof TABS)[number]["value"];
type Change = keyof typeof ACCOUNT_VIEWS;

const isTab = (v: string | null): v is Tab => TABS.some((t) => t.value === v);
const isChange = (v: string | null): v is Change => v === "username" || v === "passphrase";

const tabClass =
  "flex min-h-11 shrink-0 items-center rounded-md px-3 text-left text-[15px] font-medium whitespace-nowrap hover:bg-soft data-active:bg-soft data-active:font-semibold focus-visible:outline-offset-[-2px]";

// Approved design: a modal over the current screen with a tab for each part:
// Time and week, Calendar, Backups, Account, Sessions, About (?settings=calendar; "1" is the
// first). The tabs are a column on desktop and a scrolling row on the phone.
// Changing the username or passphrase swaps the content for that form
// (?settings=username), with a back button to the Account tab; so does
// removing a person (?settings=remove-<id>), back to the People tab.
export function SettingsDialog() {
  const [param, setView] = useModal("settings");
  const desktop = useIsDesktop();
  const showShortcuts = useShowShortcuts();
  const { logout } = useAuth();
  const notify = useNotify();
  const change = isChange(param) ? param : null;
  const { store } = useData();
  useVersion();
  const owner = store.me?.owner === true;
  // People is the owner's alone; anyone else asking for it gets the first tab.
  const tabs = TABS.filter((t) => owner || t.value !== "people");
  const asked: Tab | null = param === "1" ? "time" : isTab(param) ? param : null;
  const tab: Tab | null = asked === "people" && !owner ? "time" : asked;
  const account = change ? ACCOUNT_VIEWS[change] : null;
  const removeId = owner && param?.startsWith("remove-") ? param.slice("remove-".length) : null;
  // Leaving a swapped view puts focus back on its tab, not on the page.
  const returnTo = (t: Tab) => {
    setView(t, true);
    requestAnimationFrame(() =>
      document.querySelector<HTMLElement>("[role=dialog] [role=tab][data-active]")?.focus(),
    );
  };
  const back = () => returnTo("account");
  const toPeople = () => returnTo("people");
  const close = () => setView(null);
  const swapped = account
    ? { title: account.title, back }
    : removeId
      ? { title: "Remove a person", back: toPeople }
      : null;

  async function run(fn: () => Promise<unknown>, done?: string) {
    try {
      await fn();
      if (done) notify({ title: done, icon: CheckIcon });
    } catch (err) {
      if (err instanceof ActionError) notify({ title: err.message, icon: NoticeIcon });
      else throw err;
    }
  }

  return (
    <ResponsiveDialog
      open={tab !== null || change !== null || removeId !== null}
      onOpenChange={(o) => !o && close()}
      title={swapped?.title ?? "Settings"}
      description={account?.description}
      width={swapped ? 560 : 760}
      back={swapped ? { label: "Back to Settings", onClick: swapped.back } : undefined}
    >
      {removeId ? (
        <RemovePerson id={removeId} onDone={toPeople} />
      ) : account ? (
        <account.Form
          onClose={back}
          onSaved={() =>
            notify({
              title:
                change === "passphrase"
                  ? "Passphrase saved. Other devices were logged out."
                  : "Username saved",
              icon: CheckIcon,
            })
          }
        />
      ) : (
        tab && (
          <Tabs.Root
            value={tab}
            onValueChange={(v: Tab) => setView(v, true)}
            orientation={desktop ? "vertical" : "horizontal"}
            className="flex flex-col gap-6 min-[900px]:min-h-[440px] min-[900px]:flex-row min-[900px]:gap-8"
          >
            <div className="flex shrink-0 flex-col min-[900px]:w-44">
              <Tabs.List className="-mx-6 flex gap-1 overflow-x-auto px-6 [scrollbar-width:none] min-[900px]:mx-0 min-[900px]:flex-col min-[900px]:px-0">
                {tabs.map((t) => (
                  <Tabs.Tab key={t.value} value={t.value} className={tabClass}>
                    {t.label}
                  </Tabs.Tab>
                ))}
              </Tabs.List>
              {desktop && (
                <Button
                  variant="text"
                  className="mt-auto self-start"
                  onClick={() => {
                    close();
                    showShortcuts();
                  }}
                >
                  Keyboard shortcuts
                </Button>
              )}
            </div>
            <div className="min-w-0 flex-1">
              <Tabs.Panel value="time">
                <TimeAndWeek />
              </Tabs.Panel>
              <Tabs.Panel value="calendar">
                <Calendar run={run} />
              </Tabs.Panel>
              <Tabs.Panel value="backups">
                <Backups run={run} />
              </Tabs.Panel>
              <Tabs.Panel value="account">
                <Account onChange={(v) => setView(v)} />
              </Tabs.Panel>
              <Tabs.Panel value="sessions">
                <Sessions run={run} />
                <div className="mt-8 border-t border-line pt-6">
                  <Button variant="secondary" onClick={() => void logout()}>
                    <LogOutIcon size={20} />
                    Log out of this device
                  </Button>
                </div>
              </Tabs.Panel>
              {owner && (
                <Tabs.Panel value="people">
                  <People onRemove={(id) => setView(`remove-${id}`)} />
                </Tabs.Panel>
              )}
              <Tabs.Panel value="about">
                <About />
              </Tabs.Panel>
            </div>
          </Tabs.Root>
        )
      )}
    </ResponsiveDialog>
  );
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

function People({ onRemove }: { onRemove: (id: string) => void }) {
  const [people, reload] = usePeople();
  const actions = useSettingsActions();
  const notify = useNotify();
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
              {
                <>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-[17px]">{p.username}</p>
                    <p className="text-[13px] text-muted-foreground">
                      {p.owner
                        ? "Owner"
                        : p.mustChange
                          ? "Has not chosen a passphrase yet"
                          : "Member"}
                    </p>
                  </div>
                  {!p.owner && (
                    <Button
                      variant="text"
                      aria-label={`Remove ${p.username}`}
                      onClick={() => onRemove(p.id)}
                    >
                      Remove
                    </Button>
                  )}
                </>
              }
            </li>
          ))}
        </ul>
      )}
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

/** Confirms removing a person, in place of the Settings content. */
function RemovePerson({ id, onDone }: { id: string; onDone: () => void }) {
  const [people] = usePeople();
  const actions = useSettingsActions();
  const notify = useNotify();
  const [busy, setBusy] = useState(false);
  if (people.state === "loading") return <SkeletonRows kind="line" count={1} label="Loading" />;
  if (people.state !== "ready") return <Empty>Removing a person needs a connection.</Empty>;
  const person = people.data.find((p) => p.id === id && !p.owner);
  if (!person) {
    return (
      <>
        <Empty>This person is no longer on the server.</Empty>
        <DialogActions>
          <Button onClick={onDone}>Back to People</Button>
        </DialogActions>
      </>
    );
  }

  async function remove() {
    if (!person) return;
    setBusy(true);
    try {
      await actions.removePerson(person.id);
      notify({ title: `Removed ${person.username}`, icon: CheckIcon });
      onDone();
    } catch (err) {
      setBusy(false);
      if (err instanceof ActionError) notify({ title: err.message, icon: NoticeIcon });
      else throw err;
    }
  }

  return (
    <>
      <p className="text-[17px]">
        <span className="font-semibold">{person.username}</span> can no longer log in, and their
        planner moves to the server's removed folder. Moving it back undoes this.
      </p>
      <DialogActions>
        <Button onClick={() => void remove()} disabled={busy}>
          {busy ? "Removing" : `Remove ${person.username}`}
        </Button>
        {/* Focus starts on Cancel, so a stray Enter does nothing drastic. */}
        <Button variant="text" className="self-center" autoFocus onClick={onDone}>
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

function Account({ onChange }: { onChange: (view: keyof typeof ACCOUNT_VIEWS) => void }) {
  const { store } = useData();
  const today = useToday();
  const [account] = useAccount();
  const info = account.state === "ready" ? account.data : null;
  const username = info?.username ?? store.me?.username ?? "";

  const row = (label: string, value: string, what: keyof typeof ACCOUNT_VIEWS) => (
    <li className="flex min-h-16 items-center gap-3 border-t border-line py-2.5 first:border-t-0">
      <div className="min-w-0 flex-1">
        <p className="text-[13px] text-muted-foreground">{label}</p>
        <p className="truncate text-[17px]">{value}</p>
      </div>
      <Button
        variant="text"
        aria-label={`Change ${what}`}
        onClick={() => onChange(what)}
        disabled={account.state === "offline"}
      >
        Change
      </Button>
    </li>
  );

  return (
    <div>
      <ul>
        {row("Username", username || "Not loaded", "username")}
        {row(
          "Passphrase",
          info
            ? `Changed ${changedOn(info.passphraseChangedAt, store.me?.tz ?? "UTC", today)}`
            : "Set",
          "passphrase",
        )}
      </ul>
      {account.state === "offline" && (
        <p className="mt-1 text-[13px] text-muted-foreground">Changing these needs a connection.</p>
      )}
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
