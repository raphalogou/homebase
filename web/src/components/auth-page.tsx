import type { ReactNode } from "react";
import { Wordmark } from "./logo";

/**
 * Log in and Set up. The phone puts a serif statement in the middle and the
 * form in the thumb zone; desktop puts the statement on an ink panel at the
 * left and centres the form in the rest.
 */
export function AuthPage({
  title,
  statement,
  children,
}: {
  title: string;
  statement: string;
  children: ReactNode;
}) {
  return (
    <div className="flex min-h-dvh flex-col min-[900px]:flex-row">
      <div className="mx-auto flex w-full max-w-110 flex-1 flex-col px-6 pt-5 min-[900px]:mx-0 min-[900px]:w-[46%] min-[900px]:max-w-none min-[900px]:flex-none min-[900px]:bg-panel min-[900px]:px-14 min-[900px]:pt-14 min-[900px]:pb-14 min-[900px]:text-panel-text">
        <Wordmark className="text-[22px] min-[900px]:text-[28px]" />
        <div className="flex flex-1 flex-col justify-center py-10 min-[900px]:justify-end min-[900px]:py-0">
          <Horizon />
          <p className="whitespace-pre-line font-serif text-[36px]/[1.15] min-[900px]:text-[46px]/[1.15]">
            {statement}
          </p>
          <p className="mt-6 hidden text-panel-muted min-[900px]:block">
            A calm place for goals, projects and tasks.
          </p>
        </div>
      </div>
      <main className="mx-auto w-full max-w-[440px] px-6 pb-7 min-[900px]:flex min-[900px]:max-w-none min-[900px]:flex-1 min-[900px]:items-center min-[900px]:justify-center min-[900px]:py-14">
        <div className="min-[900px]:w-[380px]">
          <h1 className="text-[28px]/[1.2] font-bold tracking-[-0.01em]">{title}</h1>
          {children}
        </div>
      </main>
    </div>
  );
}

/** A house on a horizon with the highlighter as the sun. Decoration for the desktop panel only. */
function Horizon() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 184 96"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.4"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="my-auto hidden w-80 text-panel-muted min-[900px]:block [@media(max-height:700px)]:hidden"
    >
      <circle cx="138" cy="34" r="16" className="fill-logo-yellow" stroke="none" />
      <path d="M4 80h176M40 80V52l22-18 22 18v28M56 80V64h12v16M98 80c6-10 14-14 22-14s16 4 22 14" />
      <path d="M14 88h40M70 88h90" opacity=".5" />
    </svg>
  );
}
