import { cn } from "@/lib/utils";

/**
 * The Homebase mark: a roof over three lines, the first highlighted. The
 * roof and lines take the text colour, the highlight keeps the logo's
 * yellow in both themes. Cropped to the drawing and 1em tall, so it is as
 * tall as the text beside it.
 */
export function LogoMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="108 103 296 298"
      aria-hidden="true"
      className={cn("size-[1em] shrink-0", className)}
    >
      <path
        d="M130 220 256 125 382 220"
        fill="none"
        stroke="currentColor"
        strokeWidth="38"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <rect x="160" y="245" width="192" height="34" rx="17" className="fill-logo-yellow" />
      <rect x="160" y="305" width="144" height="34" rx="17" fill="currentColor" />
      <rect x="160" y="365" width="96" height="34" rx="17" fill="currentColor" />
    </svg>
  );
}

/** The mark beside the name, both the same height; the size comes from the font size. */
export function Wordmark({ className }: { className?: string }) {
  return (
    <p
      className={cn(
        "flex items-center gap-[0.38em] leading-none font-bold tracking-[-0.01em]",
        className,
      )}
    >
      <LogoMark />
      Homebase
    </p>
  );
}
