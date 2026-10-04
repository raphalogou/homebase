import { createContext, useContext } from "react";
import type { InstallPrompt } from "@/lib/install";
import { useInstall } from "@/lib/install";
import { Button } from "./ui/button";

export const InstallContext = createContext<InstallPrompt | null>(null);

/** A quiet "Install app" text button, shown only when the browser offers it. */
export function InstallButton({ className }: { className?: string }) {
  const prompt = useContext(InstallContext);
  if (!prompt) return null;
  return <InstallInner prompt={prompt} className={className} />;
}

function InstallInner({
  prompt,
  className,
}: {
  prompt: InstallPrompt;
  className?: string | undefined;
}) {
  const { available, install } = useInstall(prompt);
  if (!available) return null;
  return (
    <div className={className}>
      <Button variant="text" onClick={() => void install()}>
        Install app
      </Button>
    </div>
  );
}
