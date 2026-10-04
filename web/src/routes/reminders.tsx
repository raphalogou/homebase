import { useEffect, useId, useState } from "react";
import { Columns } from "@/components/app-shell";
import { CheckIcon, NoticeIcon, RemindersIcon } from "@/components/icons";
import { Empty, ScreenTitle, SectionLabel } from "@/components/section";
import { useNotify } from "@/components/toaster";
import { Button } from "@/components/ui/button";
import { ToggleSwitch } from "@/components/ui/toggle-switch";
import { ActionError } from "@/data/hooks";
import { useAdoptBrowserZone, usePush, useReminders, useSaveReminder } from "@/data/reminders";
import type { Device, Reminder } from "@/data/types";

const KINDS: Record<Reminder["kind"], { name: string; text: string }> = {
  focus: { name: "Morning focus", text: "One of your goals, and the three you chose for today." },
  checkin: { name: "Midday check", text: "How many are done, and what comes next." },
  wrap: {
    name: "Close the day",
    text: "What got done, and a nudge to move the rest or let it go.",
  },
};

export default function Reminders() {
  const reminders = useReminders();
  const zone = useAdoptBrowserZone();
  const notify = useNotify();

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
      <ScreenTitle>Reminders</ScreenTitle>
      <p className="mt-2 text-muted-foreground">
        Up to three a day, written from what you planned.
        {zone ? ` Times are in ${zone.replace(/_/g, " ")}.` : ""}
      </p>

      <ul className="mt-8">
        {reminders.map((r) => (
          <Slot key={r.slot} reminder={r} run={run} />
        ))}
      </ul>

      <ThisDevice run={run} />
    </>
  );
  return <Columns main={main} />;
}

type Run = (fn: () => Promise<unknown>, done?: string) => Promise<void>;

function Slot({ reminder, run }: { reminder: Reminder; run: Run }) {
  const save = useSaveReminder();
  const [time, setTime] = useState(reminder.atLocal);
  const id = useId();
  const kind = KINDS[reminder.kind];
  useEffect(() => setTime(reminder.atLocal), [reminder.atLocal]);

  const commit = (patch: Partial<Reminder>) =>
    run(() =>
      save({
        slot: reminder.slot,
        enabled: patch.enabled ?? reminder.enabled,
        atLocal: patch.atLocal ?? reminder.atLocal,
        kind: reminder.kind,
      }),
    );

  return (
    <li className="flex items-center gap-4 border-t border-line py-4">
      <div className="min-w-0 flex-1">
        <label htmlFor={id} className="sr-only">
          {kind.name} time
        </label>
        <input
          id={id}
          type="time"
          value={time}
          onChange={(e) => {
            // A complete time saves at once; phones pick it in a dialog.
            const v = e.target.value;
            setTime(v);
            if (/^\d{2}:\d{2}$/.test(v) && v !== reminder.atLocal) void commit({ atLocal: v });
          }}
          onBlur={() => {
            if (!/^\d{2}:\d{2}$/.test(time)) setTime(reminder.atLocal);
          }}
          className={`-ml-1 rounded-md bg-transparent px-1 text-[34px]/[1.1] font-semibold tabular-nums ${reminder.enabled ? "" : "text-muted-foreground"}`}
        />
        <p className="mt-1 font-semibold">{kind.name}</p>
        <p className="text-[13px] text-muted-foreground">{kind.text}</p>
      </div>
      <ToggleSwitch
        label={`${kind.name} at ${reminder.atLocal}`}
        checked={reminder.enabled}
        onChange={(on) => void commit({ enabled: on })}
      />
    </li>
  );
}

function since(ms: number): string {
  const mins = Math.round((Date.now() - ms) / 60000);
  if (mins < 2) return "just now";
  if (mins < 60) return `${mins} min ago`;
  const hours = Math.round(mins / 60);
  if (hours < 36) return `${hours} h ago`;
  return `${Math.round(hours / 24)} days ago`;
}

function ThisDevice({ run }: { run: Run }) {
  const { state, devices, turnOn, turnOff, remove, sendTest } = usePush();
  const notify = useNotify();
  const [busy, setBusy] = useState(false);
  const currentId = state.kind === "on" ? state.id : null;
  const others = (devices ?? []).filter((d) => d.id !== currentId);

  async function act(fn: () => Promise<unknown>, done?: string) {
    setBusy(true);
    await run(fn, done);
    setBusy(false);
  }

  return (
    <>
      <section aria-labelledby="this-device" className="mt-12">
        <SectionLabel id="this-device">This device</SectionLabel>
        <div className="border-t border-line pt-4">
          {state.kind === "loading" && (
            <p className="text-muted-foreground">Checking this device.</p>
          )}
          {state.kind === "unsupported" && (
            <p>
              This browser cannot show reminders. On Android, open Homebase in Chrome and install it
              from the menu first.
            </p>
          )}
          {state.kind === "blocked" && (
            <p>
              Notifications are blocked for Homebase. Allow them in the browser's site settings,
              then come back.
            </p>
          )}
          {state.kind === "off" && (
            <>
              <p className="mb-4 text-muted-foreground">This device does not get reminders yet.</p>
              <Button
                disabled={busy}
                onClick={() => void act(turnOn, "This device now gets reminders")}
              >
                <RemindersIcon size={20} />
                Get reminders on this device
              </Button>
            </>
          )}
          {state.kind === "on" && (
            <>
              <p className="mb-4">This device gets reminders.</p>
              <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
                <Button
                  variant="secondary"
                  disabled={busy}
                  onClick={() =>
                    void act(async () => {
                      const r = await sendTest();
                      notify({
                        title:
                          r.failed === 0
                            ? `Test sent to ${r.sent} ${r.sent === 1 ? "device" : "devices"}`
                            : `Test sent to ${r.sent}, ${r.failed} did not answer`,
                        icon: RemindersIcon,
                      });
                    })
                  }
                >
                  Send a test now
                </Button>
                <Button
                  variant="text"
                  disabled={busy}
                  onClick={() => void act(turnOff, "Reminders off on this device")}
                >
                  Turn off on this device
                </Button>
              </div>
            </>
          )}
        </div>
      </section>

      <section aria-labelledby="other-devices" className="mt-10">
        <SectionLabel id="other-devices">Other devices</SectionLabel>
        {devices === null ? (
          <Empty>The list of devices needs a connection.</Empty>
        ) : others.length === 0 ? (
          <Empty>No other devices get reminders.</Empty>
        ) : (
          <ul>
            {others.map((d) => (
              <DeviceRow
                key={d.id}
                device={d}
                onRemove={() => void act(() => remove(d.id), "Device removed")}
              />
            ))}
          </ul>
        )}
      </section>
    </>
  );
}

function DeviceRow({ device, onRemove }: { device: Device; onRemove: () => void }) {
  const name = device.label || "Unnamed device";
  return (
    <li className="flex min-h-16 items-center gap-3 border-t border-line py-2.5">
      <div className="min-w-0 flex-1">
        <p className="font-medium">{name}</p>
        <p className="text-[13px] text-muted-foreground">
          {device.lastOk
            ? `Last reached ${since(device.lastOk)}`
            : `Added ${since(device.createdAt)}`}
        </p>
      </div>
      <Button variant="text" aria-label={`Remove ${name}`} onClick={onRemove}>
        Remove
      </Button>
    </li>
  );
}
