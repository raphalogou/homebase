import { useSyncExternalStore } from "react";

function subscribeTo(query: string) {
  return (fn: () => void) => {
    const m = window.matchMedia(query);
    m.addEventListener("change", fn);
    return () => m.removeEventListener("change", fn);
  };
}

export function useMedia(query: string): boolean {
  return useSyncExternalStore(subscribeTo(query), () => window.matchMedia(query).matches);
}

/** 900 px and wider: rail and right column instead of tab bar and sheets. */
export function useIsDesktop(): boolean {
  return useMedia("(min-width: 900px)");
}
