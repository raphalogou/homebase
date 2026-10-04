import { cva, type VariantProps } from "class-variance-authority";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

// DESIGN.md: primary is ink with light text, 52 px, radius 12; secondary is
// a 1 px ink outline; text buttons are 600 with a 3 px underline offset.
const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-2 font-semibold whitespace-nowrap select-none disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        primary: "rounded-lg bg-ink text-bg",
        secondary: "rounded-lg border border-ink bg-field text-ink",
        text: "min-h-11 px-1 text-ink underline underline-offset-[3px]",
        chip: "h-11 rounded-[22px] border border-border-strong bg-field px-4 text-sm text-ink aria-pressed:border-ink aria-pressed:bg-ink aria-pressed:text-field",
        small: "h-11 rounded-md border border-border-strong bg-field px-3.5 text-sm text-ink",
        icon: "size-11 rounded-md text-ink",
      },
      size: {
        default: "",
        large: "h-[52px] px-6 text-base",
        compact: "h-11 px-4 text-sm",
      },
    },
    compoundVariants: [
      { variant: "primary", size: "default", className: "h-[52px] px-6 text-base" },
      { variant: "secondary", size: "default", className: "h-[52px] px-6 text-base" },
    ],
    defaultVariants: { variant: "primary", size: "default" },
  },
);

export type ButtonProps = ComponentProps<"button"> & VariantProps<typeof buttonVariants>;

export function Button({ className, variant, size, type = "button", ...props }: ButtonProps) {
  return (
    <button type={type} className={cn(buttonVariants({ variant, size }), className)} {...props} />
  );
}

export { buttonVariants };
