// Applies the theme chosen in Settings before the first paint, so a dark
// page never flashes light. It is a blocking file rather than an inline
// script because the Content-Security-Policy allows same-origin scripts only.
// The choice is a per-device preference, read synchronously, so it lives in
// localStorage; src/lib/theme.ts writes it under the same key.
(() => {
  let choice = "system";
  try {
    const v = localStorage.getItem("homebase-theme");
    if (v === "light" || v === "dark") choice = v;
  } catch {
    // Storage can be blocked; the system setting still applies.
  }
  const root = document.documentElement;
  if (choice === "system") root.removeAttribute("data-theme");
  else root.setAttribute("data-theme", choice);
  const media = window.matchMedia("(prefers-color-scheme: dark)");
  const paint = () => {
    const t = root.getAttribute("data-theme");
    const dark = t === "dark" || (t !== "light" && media.matches);
    const meta = document.querySelector('meta[name="theme-color"]');
    if (meta) meta.setAttribute("content", dark ? "#121514" : "#F2F3EF");
  };
  paint();
  media.addEventListener("change", paint);
  // src/lib/theme.ts calls this after a change in Settings.
  window.homebasePaintTheme = paint;
})();
