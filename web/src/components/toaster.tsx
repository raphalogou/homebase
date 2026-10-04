import { Toast } from "@base-ui/react/toast";
import { type ComponentType, useCallback } from "react";
import type { IconProps } from "./icons";
import { CloseIcon } from "./icons";

interface ToastData {
  icon: ComponentType<IconProps>;
}

export interface Notice {
  title: string;
  icon: ComponentType<IconProps>;
  /** One action, such as Undo or Open. */
  action?: { label: string; run: () => void };
}

/** Holds the toasts; wraps the app so any screen can notify. */
export function ToastProvider({ children }: { children: React.ReactNode }) {
  return (
    <Toast.Provider limit={5} timeout={4000}>
      {children}
    </Toast.Provider>
  );
}

/** Shows a short message about something that just happened. */
export function useNotify(): (n: Notice) => void {
  const manager = Toast.useToastManager();
  return useCallback(
    (n: Notice) => {
      const id = manager.add<ToastData>({
        title: n.title,
        data: { icon: n.icon },
        // A toast with an action stays a little longer, to give time to use it.
        timeout: n.action ? 6000 : 4000,
        ...(n.action
          ? {
              actionProps: {
                children: n.action.label,
                onClick: () => {
                  n.action?.run();
                  manager.close(id);
                },
              },
            }
          : {}),
      });
    },
    [manager],
  );
}

// Approved design: an ink bar with light text, an icon, radius 12, one line,
// at most five stacked. Phone: just above the tab bar. Desktop: bottom left
// of the main column. Polite to screen readers; never red.
export function Toaster() {
  const { toasts } = Toast.useToastManager();
  return (
    <Toast.Portal>
      <Toast.Viewport className="fixed inset-x-4 bottom-[calc(72px+var(--toast-lift,0px)+12px+env(safe-area-inset-bottom))] z-[60] flex flex-col-reverse gap-2 min-[900px]:right-auto min-[900px]:bottom-6 min-[900px]:left-[276px] min-[900px]:w-[420px]">
        {/* Past the limit of five, Base UI marks the oldest as limited. */}
        {toasts
          .filter((t) => !t.limited)
          .map((t) => {
            const Icon = (t.data as ToastData | undefined)?.icon;
            return (
              <Toast.Root
                key={t.id}
                toast={t}
                className="flex min-h-12 items-center gap-3 rounded-[12px] bg-ink py-1.5 pr-1.5 pl-4 text-field transition-[opacity,translate] duration-150 data-[ending-style]:translate-y-2 data-[ending-style]:opacity-0 data-[starting-style]:translate-y-2 data-[starting-style]:opacity-0"
              >
                {Icon && <Icon size={18} className="shrink-0" />}
                <Toast.Title className="min-w-0 flex-1 py-1.5 text-[15px]/[1.35] font-medium" />
                {t.actionProps && (
                  <Toast.Action className="min-h-11 shrink-0 rounded-md px-3 text-[15px] font-semibold underline underline-offset-[3px] focus-visible:outline-field" />
                )}
                <Toast.Close
                  aria-label="Dismiss"
                  className="grid size-11 shrink-0 place-items-center rounded-md opacity-80 focus-visible:outline-field"
                >
                  <CloseIcon size={18} />
                </Toast.Close>
              </Toast.Root>
            );
          })}
      </Toast.Viewport>
    </Toast.Portal>
  );
}
