import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

// shadcn/ui Kbd, restyled with the Homebase tokens.
export function Kbd({ className, ...props }: ComponentProps<"kbd">) {
  return (
    <kbd
      data-slot="kbd"
      className={cn(
        "pointer-events-none inline-grid h-5 min-w-5 select-none place-items-center rounded-[5px] border border-border-strong bg-field px-1 font-sans text-[11px] font-semibold text-muted-foreground",
        className,
      )}
      {...props}
    />
  );
}
