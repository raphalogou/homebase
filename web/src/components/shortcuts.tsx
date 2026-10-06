import { Dialog } from "@base-ui/react/dialog";
import { createContext, type ReactNode, useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { Kbd } from "@/components/ui/kbd";
import { useOpenTask } from "@/lib/open-task";
import { isTyping, SHORTCUTS, shortcutFor } from "@/lib/shortcuts";
import { useIsDesktop } from "@/lib/use-media";
import { Button } from "./ui/button";

const OpenShortcuts = createContext<() => void>(() => {});

/** Opens the list of shortcuts, for the button at the foot of Settings. */
export function useShowShortcuts(): () => void {
  return useContext(OpenShortcuts);
}

/** Puts focus in the capture field, waiting a few frames for a new screen to render it. */
function focusCapture(tries = 10) {
  const field = document.querySelector<HTMLInputElement>("[data-capture]");
  if (field) {
    field.focus();
    return;
  }
  if (tries > 0) requestAnimationFrame(() => focusCapture(tries - 1));
}

// The shortcuts work on desktop only, where there is a keyboard to use them.
export function Shortcuts({ children }: { children: ReactNode }) {
  const desktop = useIsDesktop();
  const navigate = useNavigate();
  const [taskId, openTask] = useOpenTask();
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!desktop) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.defaultPrevented || e.repeat) return;
      // An open dialog or sheet owns the keyboard, Esc included. Toasts are
      // non-modal dialogs (aria-modal="false") and must not count.
      if (
        document.querySelector(
          '[role="dialog"]:not([aria-modal="false"]), [role="alertdialog"]:not([aria-modal="false"])',
        )
      )
        return;
      if (e.key === "Escape") {
        if (taskId) {
          e.preventDefault();
          openTask(null);
        }
        return;
      }
      const s = shortcutFor({
        key: e.key,
        ctrlKey: e.ctrlKey,
        altKey: e.altKey,
        metaKey: e.metaKey,
        typing: isTyping(document.activeElement),
      });
      if (!s) return;
      e.preventDefault();
      if (s.kind === "help") setOpen(true);
      else if (s.kind === "go") navigate(s.to);
      else if (document.querySelector("[data-capture]")) focusCapture(0);
      else {
        navigate("/");
        focusCapture();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [desktop, navigate, taskId, openTask]);

  return (
    <OpenShortcuts value={() => setOpen(true)}>
      {children}
      <ShortcutsDialog open={open} onOpenChange={setOpen} />
    </OpenShortcuts>
  );
}

// A centred dialog, radius 20, 420 px wide. Base UI traps focus and returns
// it on close.
function ShortcutsDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Dialog.Root open={open} onOpenChange={(o) => onOpenChange(o)}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-veil" />
        <Dialog.Popup className="fixed top-1/2 left-1/2 z-50 max-h-[calc(100dvh-32px)] w-[min(420px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-[20px] bg-bg px-7 pt-7 pb-5">
          <Dialog.Title className="text-[22px]/[1.2] font-bold">Keyboard shortcuts</Dialog.Title>
          <Dialog.Description className="mt-1 text-sm text-muted-foreground">
            They work when you are not typing in a field.
          </Dialog.Description>
          <dl className="mt-4">
            {SHORTCUTS.map((s) => (
              <div
                key={s.keys}
                className="flex min-h-16 items-center justify-between gap-4 border-t border-line"
              >
                <dt>{s.label}</dt>
                <dd>
                  <Kbd className="h-7 min-w-7 px-1.5 text-[13px] text-ink">{s.keys}</Kbd>
                </dd>
              </div>
            ))}
          </dl>
          <div className="mt-2 flex justify-end">
            <Button variant="text" onClick={() => onOpenChange(false)}>
              Close
            </Button>
          </div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
