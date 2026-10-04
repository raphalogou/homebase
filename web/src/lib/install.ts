import { useSyncExternalStore } from "react";

// Chrome's install prompt event, which is not in the DOM typings.
interface InstallPromptEvent extends Event {
  prompt(): Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

type Listener = () => void;

/**
 * Holds the install prompt. Chrome fires the event early, often before React
 * renders, so main.tsx creates this at start.
 */
export class InstallPrompt {
  private event: InstallPromptEvent | null = null;
  private listeners = new Set<Listener>();

  constructor() {
    window.addEventListener("beforeinstallprompt", (e) => {
      e.preventDefault();
      this.event = e as InstallPromptEvent;
      this.notify();
    });
    window.addEventListener("appinstalled", () => {
      this.event = null;
      this.notify();
    });
  }

  private notify() {
    for (const fn of this.listeners) fn();
  }

  subscribe = (fn: Listener) => {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  };

  available = () => this.event !== null;

  async prompt(): Promise<void> {
    const e = this.event;
    if (!e) return;
    this.event = null;
    this.notify();
    await e.prompt();
  }
}

export function useInstall(p: InstallPrompt): { available: boolean; install: () => Promise<void> } {
  const available = useSyncExternalStore(p.subscribe, p.available);
  return { available, install: () => p.prompt() };
}
