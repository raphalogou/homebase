import type { ComponentProps } from "react";
import { cn } from "@/lib/utils";

export const fieldClass =
  "w-full rounded-lg border border-line bg-field px-4 text-base text-ink placeholder:text-muted-foreground disabled:opacity-50";

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
      <svg
        aria-hidden="true"
        viewBox="0 0 24 24"
        className="pointer-events-none absolute top-1/2 right-3 size-5 -translate-y-1/2"
        fill="none"
        stroke="currentColor"
        strokeWidth={1.8}
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <path d="m6 9 6 6 6-6" />
      </svg>
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
