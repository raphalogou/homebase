import { Dialog } from "@base-ui/react/dialog";
import type { ReactNode } from "react";
import { useIsDesktop } from "@/lib/use-media";
import { cn } from "@/lib/utils";
import { Sheet, useReturnFocus } from "./sheet";

interface ResponsiveDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  /** One muted sentence under the title. */
  description?: string | undefined;
  children: ReactNode;
  /** Desktop width in px. */
  width?: number;
}

// Approved pattern: a bottom sheet on the phone, a small centred dialog
// (radius 20) on desktop. Base UI traps focus and returns it on close.
export function ResponsiveDialog({
  open,
  onOpenChange,
  title,
  description,
  children,
  width = 440,
}: ResponsiveDialogProps) {
  const desktop = useIsDesktop();
  const returnFocus = useReturnFocus(open);
  if (!desktop) {
    return (
      <Sheet
        open={open}
        onOpenChange={onOpenChange}
        title={title}
        description={description}
        showTitle
      >
        {children}
      </Sheet>
    );
  }
  return (
    <Dialog.Root open={open} onOpenChange={(o) => onOpenChange(o)}>
      <Dialog.Portal>
        <Dialog.Backdrop className="fixed inset-0 z-40 bg-veil" />
        <Dialog.Popup
          finalFocus={returnFocus}
          style={{ width: `min(${width}px, calc(100vw - 32px))` }}
          className="fixed top-1/2 left-1/2 z-50 max-h-[calc(100dvh-32px)] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-[20px] bg-bg p-7"
        >
          <Dialog.Title
            className={cn("text-[26px]/[1.2] font-bold", description ? "mb-1" : "mb-4")}
          >
            {title}
          </Dialog.Title>
          {description && (
            <Dialog.Description className="mb-5 text-muted-foreground">
              {description}
            </Dialog.Description>
          )}
          {children}
        </Dialog.Popup>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

/** The primary action and a text Cancel: side by side on desktop, stacked on the phone. */
export function DialogActions({ children }: { children: ReactNode }) {
  return (
    <div className="mt-6 flex flex-col gap-3 pb-2 min-[900px]:flex-row-reverse min-[900px]:items-center min-[900px]:gap-4">
      {children}
    </div>
  );
}
