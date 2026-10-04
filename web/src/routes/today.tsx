import { type ServerHealth, useServerHealth } from "@/data/health";

const healthText: Record<ServerHealth, string> = {
  checking: "Checking the server.",
  ok: "Connected to the server.",
  unreachable: "Cannot reach the server.",
};

// Placeholder until Phase 2 builds the real Today screen.
export default function Today() {
  const health = useServerHealth();

  return (
    <main className="mx-auto max-w-[780px] px-6 pt-5 md:px-14 md:pt-12">
      <p className="text-lg font-bold">
        <span className="mark">Homebase</span>
      </p>
      <h1 className="mt-8 text-4xl/[1.1] font-bold tracking-[-0.02em] md:text-[46px]">Today</h1>
      <p className="mt-3 text-[13px] text-muted-foreground" aria-live="polite">
        {healthText[health]}
      </p>
    </main>
  );
}
