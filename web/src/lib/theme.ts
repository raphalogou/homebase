// The light, dark or system theme of this device. public/theme.js applies it
// before first paint; this module changes it from Settings. It is a per-device
// preference, not user data, and must be readable synchronously before the
// app loads, so it is kept in localStorage.

import { useSyncExternalStore } from "react";

export type Theme = "system" | "light" | "dark";

const KEY = "homebase-theme";

declare global {
  interface Window {
    homebasePaintTheme?: () => void;
  }
}

export function getTheme(): Theme {
  try {
    const v = localStorage.getItem(KEY);
    if (v === "light" || v === "dark") return v;
  } catch {
    // Blocked storage: the system setting applies.
  }
  return "system";
}

export function setTheme(theme: Theme): void {
  try {
    if (theme === "system") localStorage.removeItem(KEY);
    else localStorage.setItem(KEY, theme);
  } catch {
    // Still applied for this visit.
  }
  const root = document.documentElement;
  if (theme === "system") root.removeAttribute("data-theme");
  else root.setAttribute("data-theme", theme);
  window.homebasePaintTheme?.();
  window.dispatchEvent(new Event(KEY));
}

/** The current choice, kept in step between Settings and the rail's menu. */
export function useTheme(): Theme {
  return useSyncExternalStore((onChange) => {
    window.addEventListener(KEY, onChange);
    return () => window.removeEventListener(KEY, onChange);
  }, getTheme);
}
