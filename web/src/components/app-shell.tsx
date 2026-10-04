import type { ComponentType, ReactNode, SVGProps } from "react";
import { NavLink, Outlet } from "react-router";
import { useInbox, useTask } from "@/data/hooks";
import { useOpenTask } from "@/lib/open-task";
import { useIsDesktop } from "@/lib/use-media";
import { cn } from "@/lib/utils";
import {
  GoalsIcon,
  InboxIcon,
  PlanIcon,
  RemindersIcon,
  ReviewIcon,
  SettingsIcon,
  TodayIcon,
} from "./icons";
import { PickDialog } from "./pick-dialog";
import { Shortcuts } from "./shortcuts";
import { SyncBanner, SyncRailLine } from "./sync-banner";
import { TaskPanel, TaskSheet } from "./task-editor";
import { Toaster } from "./toaster";

interface NavItem {
  to: string;
  label: string;
  icon: ComponentType<SVGProps<SVGSVGElement> & { size?: number }>;
}

const NAV: NavItem[] = [
  { to: "/", label: "Today", icon: TodayIcon },
  { to: "/inbox", label: "Inbox", icon: InboxIcon },
  { to: "/plan", label: "Plan", icon: PlanIcon },
  { to: "/goals", label: "Goals", icon: GoalsIcon },
];

// The rail also lists the weekly review; the phone reaches it from Today.
const RAIL_EXTRA: NavItem[] = [{ to: "/review", label: "Weekly review", icon: ReviewIcon }];

function Badge({ count, className }: { count: number; className?: string }) {
  if (count === 0) return null;
  return (
    <span
      className={cn(
        "grid h-[18px] min-w-[18px] place-items-center rounded-full bg-ink px-1 text-[11px] font-semibold text-field",
        className,
      )}
    >
      {count > 99 ? "99+" : count}
    </span>
  );
}

export function Wordmark() {
  return (
    <p className="text-xl font-bold tracking-[-0.01em]">
      <span className="mark">Homebase</span>
    </p>
  );
}

export function AppShell() {
  const inbox = useInbox().length;
  return (
    <Shortcuts>
      <div className="min-h-dvh min-[900px]:flex">
        <nav
          aria-label="Main"
          className="sticky top-0 hidden h-dvh w-[220px] shrink-0 flex-col border-r border-line px-4 pt-10 min-[900px]:flex"
        >
          <div className="mb-8 px-3">
            <Wordmark />
          </div>
          <ul className="flex flex-col gap-1">
            {[...NAV, ...RAIL_EXTRA].map(({ to, label, icon: Icon }) => (
              <li key={to}>
                <NavLink
                  to={to}
                  end={to === "/"}
                  className={({ isActive }) =>
                    cn(
                      "flex min-h-11 items-center gap-3 rounded-md px-3 text-[15px] font-medium",
                      isActive && "bg-soft font-semibold",
                    )
                  }
                >
                  <Icon size={20} />
                  <span className="flex-1">{label}</span>
                  {to === "/inbox" && <Badge count={inbox} />}
                  {to === "/inbox" && inbox > 0 && (
                    <span className="sr-only">, {inbox} waiting</span>
                  )}
                </NavLink>
              </li>
            ))}
          </ul>
          {/* DESIGN.md: Reminders sits at the bottom of the rail; Settings (approved) just above. */}
          <div className="mt-auto mb-8 flex flex-col gap-1">
            {[
              { to: "/settings", label: "Settings", Icon: SettingsIcon },
              { to: "/reminders", label: "Reminders", Icon: RemindersIcon },
            ].map(({ to, label, Icon }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  cn(
                    "flex min-h-11 items-center gap-3 rounded-md px-3 text-[15px] font-medium",
                    isActive && "bg-soft font-semibold",
                  )
                }
              >
                <Icon size={20} />
                {label}
              </NavLink>
            ))}
            <SyncRailLine />
          </div>
        </nav>

        <div className="min-w-0 flex-1">
          <Outlet />
        </div>

        <nav
          aria-label="Main"
          className="fixed inset-x-0 bottom-0 z-20 border-t border-line bg-bg pb-[env(safe-area-inset-bottom)] min-[900px]:hidden"
        >
          <ul className="grid h-[72px] grid-cols-4">
            {NAV.map(({ to, label, icon: Icon }) => (
              <li key={to}>
                <NavLink
                  to={to}
                  end={to === "/"}
                  className={({ isActive }) =>
                    cn(
                      "relative flex h-full flex-col items-center justify-center gap-1 border-t-2 border-transparent text-xs font-medium text-muted-foreground",
                      isActive && "border-ink font-semibold text-ink",
                    )
                  }
                >
                  <span className="relative">
                    <Icon size={24} />
                    {to === "/inbox" && (
                      <Badge count={inbox} className="absolute -top-1.5 -right-3" />
                    )}
                  </span>
                  {label}
                  {to === "/inbox" && inbox > 0 && (
                    <span className="sr-only">, {inbox} waiting</span>
                  )}
                </NavLink>
              </li>
            ))}
          </ul>
        </nav>

        <TaskSheet />
        <PickDialog />
        <Toaster />
      </div>
    </Shortcuts>
  );
}

interface ColumnsProps {
  main: ReactNode;
  /** Desktop only: the right column. An open task takes its place. */
  side?: ReactNode;
  /** The phone needs room for the capture bar above the tab bar. */
  capture?: boolean;
}

// DESIGN.md "Layout": main column capped at 780 px; the right column is
// 300 to 380 px beside it from 1100 px, under it from 900 px.
export function Columns({ main, side, capture = false }: ColumnsProps) {
  const desktop = useIsDesktop();
  const [id] = useOpenTask();
  const task = useTask(id);
  const right = desktop ? task ? <TaskPanel /> : side : null;
  // The regions fill the window; only the content inside the main column is
  // capped, so lines stay short however wide the screen.
  return (
    <div className="min-[1100px]:flex">
      <main
        className={cn(
          "min-w-0 flex-1 px-6 pt-5 min-[900px]:px-14 min-[900px]:pt-12 min-[900px]:pb-16",
          capture ? "pb-[168px]" : "pb-[104px]",
        )}
      >
        <SyncBanner />
        <div className="mx-auto w-full max-w-[780px]">{main}</div>
      </main>
      {right && (
        <aside className="border-line px-6 pb-16 min-[900px]:px-14 min-[1100px]:sticky min-[1100px]:top-0 min-[1100px]:h-dvh min-[1100px]:w-[360px] min-[1100px]:shrink-0 min-[1100px]:overflow-y-auto min-[1100px]:border-l min-[1100px]:px-8 min-[1100px]:pt-12">
          <div className="mx-auto w-full max-w-[780px] min-[1100px]:max-w-none">{right}</div>
        </aside>
      )}
    </div>
  );
}
