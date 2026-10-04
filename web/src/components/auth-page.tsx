import type { ReactNode } from "react";
import { Wordmark } from "./app-shell";
import { ScreenTitle } from "./section";

// Log in and Set up share the top-aligned column of the other screens.
export function AuthPage({
  title,
  intro,
  children,
}: {
  title: string;
  intro: string;
  children: ReactNode;
}) {
  return (
    <main className="mx-auto w-full max-w-[400px] px-6 pt-5 pb-12 min-[900px]:pt-24">
      <Wordmark />
      <ScreenTitle className="mt-7">{title}</ScreenTitle>
      <p className="mt-2.5 text-muted-foreground">{intro}</p>
      {children}
    </main>
  );
}
