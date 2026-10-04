/** Accepts http, https and mailto links, adding https:// to a bare address. */
export function normaliseUrl(input: string): string | null {
  const s = input.trim();
  if (!s) return null;
  const withScheme = /^[a-z][a-z0-9+.-]*:/i.test(s) ? s : `https://${s}`;
  try {
    const u = new URL(withScheme);
    if ((u.protocol === "http:" || u.protocol === "https:") && u.hostname.includes("."))
      return u.href;
    if (u.protocol === "mailto:" && u.pathname.includes("@")) return u.href;
  } catch {
    return null;
  }
  return null;
}
