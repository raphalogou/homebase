// Desktop keyboard shortcuts (DESIGN.md "Keyboard shortcuts"). Single keys
// act only when the person is not typing and holds no Ctrl, Alt or Meta.

export type Shortcut = { kind: "capture" } | { kind: "go"; to: string } | { kind: "help" };

export const SHORTCUTS: { keys: string; label: string; shortcut: Shortcut | null }[] = [
  { keys: "C", label: "Capture a task", shortcut: { kind: "capture" } },
  { keys: "T", label: "Go to Today", shortcut: { kind: "go", to: "/" } },
  { keys: "I", label: "Go to Inbox", shortcut: { kind: "go", to: "/inbox" } },
  { keys: "P", label: "Go to Plan", shortcut: { kind: "go", to: "/plan" } },
  { keys: "G", label: "Go to Goals", shortcut: { kind: "go", to: "/goals" } },
  { keys: "R", label: "Weekly review", shortcut: { kind: "go", to: "/review" } },
  { keys: "S", label: "Settings", shortcut: { kind: "go", to: "/settings" } },
  { keys: "?", label: "Show this list", shortcut: { kind: "help" } },
  // Esc is handled by each dialog, sheet and the task panel.
  { keys: "Esc", label: "Close a dialog, sheet or the task panel", shortcut: null },
];

export interface KeyInput {
  key: string;
  ctrlKey: boolean;
  altKey: boolean;
  metaKey: boolean;
  /** Focus is in an input, textarea, select or editable element. */
  typing: boolean;
}

export function shortcutFor(e: KeyInput): Shortcut | null {
  if (e.typing || e.ctrlKey || e.altKey || e.metaKey) return null;
  const key = e.key === "?" ? "?" : e.key.toUpperCase();
  return SHORTCUTS.find((s) => s.keys === key)?.shortcut ?? null;
}

/** Whether keys typed at el belong to a field rather than to the shortcuts. */
export function isTyping(el: Element | null): boolean {
  if (!el) return false;
  const tag = el.tagName;
  if (tag === "TEXTAREA" || tag === "SELECT") return true;
  if (tag === "INPUT") {
    const type = (el as HTMLInputElement).type;
    return !["checkbox", "radio", "button", "submit", "reset", "range", "file"].includes(type);
  }
  return (el as HTMLElement).isContentEditable === true;
}
