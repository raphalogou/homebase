import { useId, useMemo, useState } from "react";
import { ChangePassphraseDialog, ChangeUsernameDialog } from "@/components/account-dialogs";
import { Columns } from "@/components/app-shell";
import { CheckIcon, NoticeIcon } from "@/components/icons";
import { Empty, ScreenTitle, SectionLabel } from "@/components/section";
import { useNotify } from "@/components/toaster";
import { Button } from "@/components/ui/button";
import { Input, Label, Select } from "@/components/ui/input";
import { Segmented } from "@/components/ui/segmented";
import { ActionError, useAuth, useToday, useWeekStart } from "@/data/hooks";
import { useData } from "@/data/provider";
import { useAccount, useServerSettings, useSessions, useSettingsActions } from "@/data/settings";
import type { AccountInfo, SessionInfo } from "@/data/types";
import { changedOn } from "@/lib/dates";
import { labelFromUA } from "@/lib/ua";

function ago(ms: number): string {
  const mins = Math.round((Date.now() - ms) / 60000);
  if (mins < 2) return "just now";
  if (mins < 60) return `${mins} min ago`;
  const hours = Math.round(mins / 60);
  if (hours < 36) return `${hours} h ago`;
  return `${Math.round(hours / 24)} days ago`;
}

// Approved design: one plain page with Time and week, Calendar, Sessions,
// then logging out of this device.
export default function Settings() {
  const notify = useNotify();
  const { logout } = useAuth();
  // A new passphrase logs the other devices out, so the list loads again.
  const [sessionsKey, setSessionsKey] = useState(0);

  async function run(fn: () => Promise<unknown>, done?: string) {
    try {
      await fn();
      if (done) notify({ title: done, icon: CheckIcon });
    } catch (err) {
      if (err instanceof ActionError) notify({ title: err.message, icon: NoticeIcon });
      else throw err;
    }
  }

  const main = (
    <>
      <ScreenTitle>Settings</ScreenTitle>
      <TimeAndWeek run={run} />
      <Calendar run={run} />
      <Account onSessionsChanged={() => setSessionsKey((k) => k + 1)} />
      <Sessions key={sessionsKey} run={run} />
      <div className="mt-10 border-t border-line pt-6">
        <Button variant="secondary" onClick={() => void logout()}>
          Log out of this device
        </Button>
      </div>
    </>
  );
  return <Columns main={main} />;
}

type Run = (fn: () => Promise<unknown>, done?: string) => Promise<void>;

function TimeAndWeek({ run }: { run: Run }) {
  const { store } = useData();
  const weekStart = useWeekStart();
  const actions = useSettingsActions();
  const id = useId();
  const tz = store.me?.tz ?? "UTC";
  const zones = useMemo(() => {
    const all = Intl.supportedValuesOf("timeZone");
    return all.includes(tz) ? all : [tz, ...all];
  }, [tz]);

  return (
    <section aria-labelledby={`${id}-h`} className="mt-8">
      <SectionLabel id={`${id}-h`}>Time and week</SectionLabel>
      <div className="flex flex-col gap-4 border-t border-line pt-4">
        <div>
          <Label htmlFor={`${id}-tz`}>Time zone</Label>
          <Select
            id={`${id}-tz`}
            value={tz}
            onChange={(e) =>
              void run(() => actions.saveTime(e.target.value, weekStart), "Time zone saved")
            }
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
            onChange={(v) => void run(() => actions.saveTime(tz, v === "1" ? 1 : 0))}
          />
        </div>
      </div>
    </section>
  );
}

function Calendar({ run }: { run: Run }) {
  const [settings] = useServerSettings();
  const actions = useSettingsActions();
  const [url, setUrl] = useState<string | null>(null);
  const id = useId();
  const shown = url ?? (settings.state === "ready" ? settings.data.calendarUrl : null);

  return (
    <section aria-labelledby={`${id}-h`} className="mt-10">
      <SectionLabel id={`${id}-h`}>Calendar</SectionLabel>
      <div className="border-t border-line pt-4">
        <p className="text-muted-foreground">
          Add this private link to Google Calendar ("From URL") to see due and planned tasks there.
          Google refreshes it every few hours, so changes show up slowly.
        </p>
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
    </section>
  );
}

function Account({ onSessionsChanged }: { onSessionsChanged: () => void }) {
  const { store } = useData();
  const notify = useNotify();
  const today = useToday();
  const [account] = useAccount();
  const [saved, setSaved] = useState<AccountInfo | null>(null);
  const [editing, setEditing] = useState<"username" | "passphrase" | null>(null);
  const id = useId();
  const info = saved ?? (account.state === "ready" ? account.data : null);
  const username = info?.username ?? store.me?.username ?? "";

  const row = (label: string, value: string, what: "username" | "passphrase") => (
    <li className="flex min-h-16 items-center gap-3 border-t border-line py-2.5">
      <div className="min-w-0 flex-1">
        <p className="text-[13px] text-muted-foreground">{label}</p>
        <p className="truncate text-[17px]">{value}</p>
      </div>
      <Button
        variant="text"
        aria-label={`Change ${what}`}
        onClick={() => setEditing(what)}
        disabled={account.state === "offline"}
      >
        Change
      </Button>
    </li>
  );

  return (
    <section aria-labelledby={`${id}-h`} className="mt-10">
      <SectionLabel id={`${id}-h`}>Account</SectionLabel>
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
      <ChangeUsernameDialog
        open={editing === "username"}
        onOpenChange={(o) => setEditing(o ? "username" : null)}
        onSaved={(i) => {
          setSaved(i);
          notify({ title: "Username saved", icon: CheckIcon });
        }}
      />
      <ChangePassphraseDialog
        open={editing === "passphrase"}
        onOpenChange={(o) => setEditing(o ? "passphrase" : null)}
        onSaved={(i) => {
          setSaved(i);
          onSessionsChanged();
          notify({ title: "Passphrase saved. Other devices were logged out.", icon: CheckIcon });
        }}
      />
    </section>
  );
}

function Sessions({ run }: { run: Run }) {
  const [sessions, reload] = useSessions();
  const actions = useSettingsActions();
  const id = useId();
  return (
    <section aria-labelledby={`${id}-h`} className="mt-10">
      <SectionLabel id={`${id}-h`}>Sessions</SectionLabel>
      {sessions.state === "offline" && <Empty>The list of sessions needs a connection.</Empty>}
      {sessions.state === "ready" && (
        <ul>
          {sessions.data.map((s: SessionInfo) => (
            <li key={s.id} className="flex min-h-16 items-center gap-3 border-t border-line py-2.5">
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
    </section>
  );
}
