import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

/** "Goals", "Tasks": 14 px, 600, muted, sentence case. */
export function SectionLabel({
  children,
  className,
  id,
}: {
  children: ReactNode;
  className?: string;
  id?: string;
}) {
  return (
    <h2 id={id} className={cn("mb-2 text-sm font-semibold text-muted-foreground", className)}>
      {children}
    </h2>
  );
}

/** "Your three": 22 to 24 px, 700. */
export function SectionHeading({
  children,
  className,
  id,
}: {
  children: ReactNode;
  className?: string;
  id?: string;
}) {
  return (
    <h2 id={id} className={cn("mb-3 text-[22px]/[1.2] font-bold md:text-2xl", className)}>
      {children}
    </h2>
  );
}

/** Screen title: 36/1.1, 700, -0.02em; 46 on desktop. */
export function ScreenTitle({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <h1
      className={cn(
        "text-4xl/[1.1] font-bold tracking-[-0.02em] min-[900px]:text-[46px]",
        className,
      )}
    >
      {children}
    </h1>
  );
}

/** A quiet sentence for empty lists, telling what to do. */
export function Empty({ children }: { children: ReactNode }) {
  return <p className="border-t border-line py-4 text-muted-foreground">{children}</p>;
}
