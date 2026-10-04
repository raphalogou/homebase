import { Dialog } from "@base-ui/react/dialog";
import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface SheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  /** Show the title visibly; otherwise it is only for screen readers. */
  showTitle?: boolean;
  children: ReactNode;
  className?: string;
}

// The phone's bottom sheet (DESIGN.md "Layout"): grab bar, 20 px top radius,
// slides up in 180 ms, no motion with reduced motion.
export function Sheet({
  open,
  onOpenChange,
  title,
  showTitle = false,
  children,
  className,
}: SheetProps) {
  return (
    <Dialog.Root open={open} onOpenChange={(next) => onOpenChange(next)}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-ink/30 transition-opacity duration-[180ms] data-[ending-style]:opacity-0 data-[starting-style]:opacity-0" />
        <Dialog.Popup
          className={cn(
            "fixed inset-x-0 bottom-0 z-50 flex max-h-[92dvh] flex-col rounded-t-[20px] bg-bg px-6 pb-[max(24px,env(safe-area-inset-bottom))] transition-transform duration-[180ms] ease-out data-[ending-style]:translate-y-full data-[starting-style]:translate-y-full",
            className,
          )}
        >
          <div
            aria-hidden="true"
            className="mx-auto mt-2.5 mb-3 h-1 w-10 shrink-0 rounded-full bg-dashed"
          />
          <Dialog.Title className={showTitle ? "mb-4 text-[30px]/[1.15] font-bold" : "sr-only"}>
            {title}
          </Dialog.Title>
          <div className="-mx-6 overflow-y-auto px-6">{children}</div>
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
