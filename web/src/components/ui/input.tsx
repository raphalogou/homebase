import { ChevronDown } from "lucide-react";
import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

export const fieldClass =
  "w-full rounded-sm border border-border-strong bg-field px-4 text-base text-ink placeholder:text-muted-foreground focus-visible:border-ink focus-visible:ring-[3px] focus-visible:ring-ink/20 focus-visible:outline-none disabled:opacity-50";

export function Input({ className, ...props }: ComponentProps<"input">) {
  return <input className={cn(fieldClass, "h-12", className)} {...props} />;
}

export function Textarea({ className, ...props }: ComponentProps<"textarea">) {
  return (
    <textarea className={cn(fieldClass, "min-h-32 py-3 leading-[1.55]", className)} {...props} />
  );
}

/** A native select: on a phone it opens the system picker, which beats any custom list. */
export function Select({ className, ...props }: ComponentProps<"select">) {
  return (
    <span className="relative block">
      <select className={cn(fieldClass, "h-12 appearance-none pr-10", className)} {...props} />
      <ChevronDown
        aria-hidden="true"
        size={20}
        strokeWidth={1.8}
        className="pointer-events-none absolute top-1/2 right-3 -translate-y-1/2"
      />
    </span>
  );
}

export function Label({
  className,
  htmlFor,
  children,
  ...props
}: ComponentProps<"label"> & { htmlFor: string }) {
  return (
    <label
      htmlFor={htmlFor}
      className={cn("mb-2 block text-sm font-semibold text-muted-foreground", className)}
      {...props}
    >
      {children}
    </label>
  );
}
