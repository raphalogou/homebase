import type { ReactNode, SVGProps } from "react";

// Icons are drawn inline (DESIGN.md): 24 px, 1.8 px stroke, round caps.
type IconProps = SVGProps<SVGSVGElement> & { size?: number };

function Icon({ size = 24, children, ...props }: IconProps & { children: ReactNode }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      {...props}
    >
      {children}
    </svg>
  );
}

export function TodayIcon(p: IconProps) {
  return (
    <Icon {...p}>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 3v2M12 19v2M3 12h2M19 12h2M5.6 5.6l1.4 1.4M17 17l1.4 1.4M5.6 18.4 7 17M17 7l1.4-1.4" />
    </Icon>
  );
}

export function InboxIcon(p: IconProps) {
  return (
    <Icon {...p}>
      <path d="M4 13h4.5l1.5 2.5h4l1.5-2.5H20" />
      <path d="M6 5h12l2 8v5a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1v-5z" />
    </Icon>
  );
}

export function PlanIcon(p: IconProps) {
  return (
    <Icon {...p}>
      <rect x="4" y="5" width="16" height="15" rx="2" />
      <path d="M4 10h16M9 3v4M15 3v4" />
    </Icon>
  );
}

export function GoalsIcon(p: IconProps) {
  return (
    <Icon {...p}>
      <path d="M6 21V4M6 4h11l-2 4 2 4H6" />
    </Icon>
  );
}

export function GripIcon(p: IconProps) {
  return (
    <Icon {...p}>
      <path d="M9 6h.01M15 6h.01M9 12h.01M15 12h.01M9 18h.01M15 18h.01" strokeWidth={2.6} />
    </Icon>
  );
}

export function CheckIcon(p: IconProps) {
  return (
    <Icon {...p}>
      <path d="m6 12.5 4 4 8-9" />
    </Icon>
  );
}

export function CloseIcon(p: IconProps) {
  return (
    <Icon {...p}>
      <path d="M6 6l12 12M18 6 6 18" />
    </Icon>
  );
}

export function UpIcon(p: IconProps) {
  return (
    <Icon {...p}>
      <path d="M12 19V5M6 11l6-6 6 6" />
    </Icon>
  );
}

export function DownIcon(p: IconProps) {
  return (
    <Icon {...p}>
      <path d="M12 5v14M6 13l6 6 6-6" />
    </Icon>
  );
}

export function RepeatIcon(p: IconProps) {
  return (
    <Icon {...p}>
      <path d="M4 11V9a3 3 0 0 1 3-3h12M16 3l3 3-3 3M20 13v2a3 3 0 0 1-3 3H5M8 21l-3-3 3-3" />
    </Icon>
  );
}
