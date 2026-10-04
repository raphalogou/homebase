import { Link } from "react-router";
import { BackIcon } from "./icons";

/** Detail screens go back with a text link at the top left (DESIGN.md "Layout"). */
export function BackLink({ to, children }: { to: string; children: string }) {
  return (
    <Link
      to={to}
      className="-ml-1.5 inline-flex min-h-11 max-w-full items-center gap-1 font-semibold"
    >
      <BackIcon size={20} />
      <span className="truncate underline underline-offset-[3px]">{children}</span>
    </Link>
  );
}
