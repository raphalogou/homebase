import type { ComponentType } from "react";
import type { SyncIndicator } from "@/data/sync-indicator";
import { useFirstLoad, useSyncIndicator } from "@/data/sync-status";
import { useIsDesktop } from "@/lib/use-media";
import { type IconProps, NoticeIcon, OfflineIcon, SyncedIcon, SyncingIcon } from "./icons";
import { Button } from "./ui/button";

interface Message {
  icon: ComponentType<IconProps>;
  lead: string;
  text: string;
}

function message(i: SyncIndicator, device: string): Message | null {
  switch (i.kind) {
    case "offline":
      return {
        icon: OfflineIcon,
        lead: "Offline",
        text: `Changes are saved on this ${device} and will sync when you reconnect.${i.waiting ? ` ${i.waiting} waiting.` : ""}`,
      };
    case "error":
      return {
        icon: NoticeIcon,
        lead: "Could not sync",
        text: `Your changes are saved here. ${
          i.retryIn > 0
            ? `Trying again in ${i.retryIn} second${i.retryIn === 1 ? "" : "s"}.`
            : "Trying again now."
        }`,
      };
    case "syncing":
      return { icon: SyncingIcon, lead: "Syncing", text: "" };
    case "recovered":
      return { icon: SyncedIcon, lead: "Back online. Synced.", text: "" };
    default:
      return null;
  }
}

/**
 * The banner at the top of the main column. It pushes the page down rather
 * than covering it, and is announced politely. Nothing shows when synced.
 */
export function SyncBanner() {
  const { indicator, retry } = useSyncIndicator();
  const desktop = useIsDesktop();
  const { loading } = useFirstLoad();
  // With nothing on screen yet, the page's own error block explains instead.
  const m = loading ? null : message(indicator, desktop ? "computer" : "phone");
  return (
    <div role="status" aria-live="polite">
      {m && (
        <div className="-mx-6 -mt-5 mb-5 flex items-start gap-3 bg-soft px-6 py-3 min-[900px]:-mx-14 min-[900px]:-mt-12 min-[900px]:mb-8 min-[900px]:items-center min-[900px]:px-14">
          <m.icon size={18} className="mt-[3px] shrink-0 min-[900px]:mt-0" />
          <p className="min-w-0 flex-1 text-[15px]">
            <strong className="font-semibold max-[899px]:block">
              {m.lead}
              {desktop && m.text && !m.lead.endsWith(".") ? "." : ""}
            </strong>
            {m.text && " "}
            {m.text && <span className="text-muted-foreground">{m.text}</span>}
          </p>
          {indicator.kind === "error" && (
            <Button variant="text" className="-my-2 shrink-0" onClick={retry}>
              Retry
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

/** One quiet line under Reminders in the rail; the banner does the announcing. */
export function SyncRailLine() {
  const { indicator } = useSyncIndicator();
  let text = "";
  if (indicator.kind === "offline")
    text = indicator.waiting ? `Offline, ${indicator.waiting} waiting` : "Offline";
  else if (indicator.kind === "error") text = "Could not sync";
  else if (indicator.kind === "syncing") text = "Syncing";
  else if (indicator.kind === "recovered") text = "Synced";
  if (!text) return null;
  return (
    <p aria-hidden="true" className="px-3 pt-2 text-[13px] text-muted-foreground">
      {text}
    </p>
  );
}
