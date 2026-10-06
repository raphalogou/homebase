import { Dialog } from "@base-ui/react/dialog";
import { type ReactNode, type RefObject, useRef } from "react";
import { cn } from "@/lib/utils";

interface SheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  /** Show the title visibly; otherwise it is only for screen readers. */
  showTitle?: boolean;
  /** One muted sentence under a shown title. */
  description?: string | undefined;
  children: ReactNode;
  className?: string;
}

/**
 * The element that had focus when a dialog opened, for Base UI to return
 * focus to on close. Its own guess only knows Dialog.Trigger, and ours are
 * opened by plain buttons. Read during the render that opens it, before
 * focus moves into the dialog.
 */
export function useReturnFocus(open: boolean): RefObject<HTMLElement | null> {
  const ref = useRef<HTMLElement | null>(null);
  const wasOpen = useRef(false);
  if (open && !wasOpen.current && document.activeElement instanceof HTMLElement) {
    ref.current = document.activeElement;
  }
  wasOpen.current = open;
  return ref;
}

// The phone's bottom sheet (DESIGN.md "Layout"): grab bar, 20 px top radius,
// slides up in 180 ms, no motion with reduced motion.
export function Sheet({
  open,
  onOpenChange,
  title,
  showTitle = false,
  description,
  children,
  className,
}: SheetProps) {
  const returnFocus = useReturnFocus(open);
  return (
    <Dialog.Root open={open} onOpenChange={(next) => onOpenChange(next)}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-veil transition-opacity duration-[180ms] data-[ending-style]:opacity-0 data-[starting-style]:opacity-0" />
        <Dialog.Popup
          finalFocus={returnFocus}
          className={cn(
            "fixed inset-x-0 bottom-0 z-50 flex max-h-[92dvh] flex-col rounded-t-[20px] bg-bg px-6 pb-[max(24px,env(safe-area-inset-bottom))] transition-transform duration-[180ms] ease-out data-[ending-style]:translate-y-full data-[starting-style]:translate-y-full",
            className,
          )}
        >
          <div
            aria-hidden="true"
            className="mx-auto mt-2.5 mb-3 h-1 w-10 shrink-0 rounded-full bg-border-strong"
          />
          <Dialog.Title
            className={cn(
              showTitle ? "text-[30px]/[1.15] font-bold" : "sr-only",
              showTitle && (description ? "mb-1" : "mb-4"),
            )}
          >
            {title}
          </Dialog.Title>
          {description && (
            <Dialog.Description className="mb-5 text-muted-foreground">
              {description}
            </Dialog.Description>
          )}
          <div className="-mx-6 overflow-y-auto px-6">{children}</div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
