import { useEffect, useState } from "react";

export type ServerHealth = "checking" | "ok" | "unreachable";

/** Reports whether the Go server answers, and checks again when the device comes back online. */
export function useServerHealth(): ServerHealth {
  const [health, setHealth] = useState<ServerHealth>("checking");

  useEffect(() => {
    let controller = new AbortController();

    const check = async () => {
      controller.abort();
      controller = new AbortController();
      try {
        const res = await fetch("/healthz", { signal: controller.signal, cache: "no-store" });
        setHealth(res.ok ? "ok" : "unreachable");
      } catch (err) {
        if (!(err instanceof DOMException && err.name === "AbortError")) {
          setHealth("unreachable");
        }
      }
    };

    void check();
    window.addEventListener("online", check);
    return () => {
      window.removeEventListener("online", check);
      controller.abort();
    };
  }, []);

  return health;
}
