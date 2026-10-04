import { Menu } from "@base-ui/react/menu";
import type { ComponentProps } from "react";
import { CheckIcon } from "@/components/icons";
import { cn } from "@/lib/utils";

// shadcn/ui DropdownMenu on Base UI, only the parts Homebase uses.
export const DropdownMenu = Menu.Root;
export const DropdownMenuTrigger = Menu.Trigger;
export const DropdownMenuRadioGroup = Menu.RadioGroup;

export function DropdownMenuContent({
  className,
  side = "bottom",
  align = "start",
  ...props
}: ComponentProps<typeof Menu.Popup> & {
  side?: "top" | "bottom" | "left" | "right";
  align?: "start" | "center" | "end";
}) {
  return (
    <Menu.Portal>
      <Menu.Positioner side={side} align={align} sideOffset={6} className="z-50">
        <Menu.Popup
          className={cn(
            "min-w-44 rounded-lg border border-line bg-field p-1 text-ink shadow-lift outline-none",
            className,
          )}
          {...props}
        />
      </Menu.Positioner>
    </Menu.Portal>
  );
}

export function DropdownMenuRadioItem({
  className,
  children,
  ...props
}: ComponentProps<typeof Menu.RadioItem>) {
  return (
    <Menu.RadioItem
      className={cn(
        "flex min-h-11 cursor-pointer items-center gap-3 rounded-md px-3 text-[15px] font-medium outline-none select-none data-highlighted:bg-soft",
        className,
      )}
      {...props}
    >
      {children}
      <Menu.RadioItemIndicator className="ml-auto">
        <CheckIcon size={18} />
      </Menu.RadioItemIndicator>
    </Menu.RadioItem>
  );
}
