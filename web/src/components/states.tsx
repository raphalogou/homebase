import type { ComponentProps, ReactNode } from "react";
import { Link } from "react-router";
import { useFirstLoad } from "@/data/sync-status";
import { cn } from "@/lib/utils";
import { NoticeIcon } from "./icons";
import { Button } from "./ui/button";

// Empty, loading and error states (DESIGN.md "States"). Empty says what the
// thing is and offers one action; loading shows skeleton bars only on a first
// load; an error is an icon, a bold title and plain words, in ink.

type Action = ({ label: string; onClick: () => void } | { label: string; to: string }) & {
  /** A quieter text link instead of the outlined button. */
  text?: boolean;
};

function ActionButton({ action }: { action: Action }) {
  const cls = "mt-4 h-12 px-5 text-base";
  if ("to" in action)
    return (
      <Link
        to={action.to}
        className={cn(
          "inline-flex items-center justify-center font-semibold",
          action.text
            ? "mt-2 min-h-11 underline underline-offset-[3px]"
            : cn("rounded-lg border border-ink bg-field", cls),
        )}
      >
        {action.label}
      </Link>
    );
  if (action.text)
    return (
      <Button variant="text" className="mt-2" onClick={action.onClick}>
        {action.label}
      </Button>
    );
  return (
    <Button variant="secondary" className={cls} onClick={action.onClick}>
      {action.label}
    </Button>
  );
}

/** Nothing here yet: a bold title, a sentence, and at most one action. */
export function EmptyBlock({
  title,
  children,
  action,
  serif = false,
  className,
}: {
  title: string;
  children?: ReactNode;
  action?: Action | undefined;
  /** Goals read as statements of intent, so their empty title is in the serif. */
  serif?: boolean;
  className?: string;
}) {
  return (
    <div className={cn("border-t border-line pt-5 pb-2", className)}>
      <p className={serif ? "font-serif text-xl/[1.25]" : "text-[17px] font-bold"}>{title}</p>
      {children && <p className="mt-1 text-muted-foreground">{children}</p>}
      {action && <ActionButton action={action} />}
    </div>
  );
}

/** Something failed and there is nothing to show instead. */
export function ErrorBlock({
  title,
  children = "Check your connection, then try again. Nothing has been lost.",
  action,
  className,
}: {
  title: string;
  children?: ReactNode;
  action?: Action;
  className?: string;
}) {
  return (
    <div className={cn("border-t border-line pt-5 pb-2", className)} role="alert">
      <p className="flex items-center gap-2.5 text-xl/[1.3] font-bold">
        <NoticeIcon size={20} className="shrink-0" />
        {title}
      </p>
      <p className="mt-1 text-muted-foreground">{children}</p>
      {action && <ActionButton action={action} />}
    </div>
  );
}

function Bar({ className }: { className?: string }) {
  return (
    <span className={cn("block h-5 rounded-md bg-soft motion-safe:animate-pulse", className)} />
  );
}

/** Placeholder rows for a first load: task rows with a circle, or plain lines. */
export function SkeletonRows({
  count = 2,
  kind = "task",
  label = "Loading",
  bare = false,
}: {
  count?: number;
  kind?: "task" | "line";
  label?: string;
  /** Inside another skeleton, which already announces the loading. */
  bare?: boolean;
}) {
  return (
    <div {...(bare ? {} : { "aria-busy": true, "aria-label": label, role: "status" })}>
      {!bare && <span className="sr-only">{label}</span>}
      {Array.from({ length: count }, (_, i) => `row-${i}`).map((key, i) => (
        <div
          key={key}
          className="flex min-h-[72px] items-center gap-4 border-t border-line py-3"
          aria-hidden="true"
        >
          {kind === "task" && <span className="size-[26px] shrink-0 rounded-full bg-soft" />}
          <span className="flex-1">
            <Bar className={i % 2 ? "w-3/5" : "w-3/4"} />
            <Bar className="mt-2 h-3 w-2/5" />
          </span>
        </div>
      ))}
    </div>
  );
}

/** A detail screen's first load: a title bar, a progress line and rows. */
export function DetailSkeleton({ what }: { what: string }) {
  return (
    <div aria-busy="true" role="status" aria-label={`Loading the ${what}`}>
      <span className="sr-only">{`Loading the ${what}`}</span>
      <div aria-hidden="true">
        <Bar className="h-9 w-2/3" />
        <Bar className="mt-5 h-1 w-full" />
        <Bar className="mt-2 h-3 w-1/4" />
        <div className="mt-10">
          <SkeletonRows bare />
        </div>
      </div>
    </div>
  );
}

/**
 * Shows children once this device has its data. Until the first sync after
 * logging in it shows the skeleton; if that sync fails, the error block.
 * Afterwards the local copy always shows at once, even offline.
 */
export function FirstLoad({
  skeleton,
  what,
  children,
}: {
  skeleton: ReactNode;
  /** "tasks", "projects", "goals": used in "Could not load your tasks". */
  what: string;
  children: ReactNode;
}) {
  const { loading, failed, retry } = useFirstLoad();
  if (failed)
    return (
      <ErrorBlock
        title={`Could not load your ${what}`}
        action={{ label: "Try again", onClick: retry }}
      />
    );
  if (loading) return <>{skeleton}</>;
  return <>{children}</>;
}

/** A plain line for a failed save, under the control it belongs to. */
export function InlineError({ children, ...props }: ComponentProps<"p">) {
  return (
    <p className="mt-1.5 flex items-start gap-2 text-[15px] text-ink" role="alert" {...props}>
      <NoticeIcon size={18} className="mt-[2px] shrink-0" />
      {children}
    </p>
  );
}
