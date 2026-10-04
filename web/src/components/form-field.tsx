import { type ComponentProps, useId, useLayoutEffect, useRef, useState } from "react";
import { Input, Label } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { HideIcon, ShowIcon, WarningIcon } from "./icons";

interface FieldProps extends Omit<ComponentProps<"input">, "id" | "type"> {
  label: string;
  /** A quiet line under the field, such as the passphrase rule. */
  hint?: string | undefined;
  /** Shown under the field in --danger, with a 2 px --danger border on the field. */
  error?: string | undefined;
}

// The owner asked for errors in --danger: the border, the outline, the text and its icon.
function Messages({
  id,
  hint,
  error,
}: {
  id: string;
  hint?: string | undefined;
  error?: string | undefined;
}) {
  return (
    <>
      {hint && (
        <p id={`${id}-hint`} className="mt-1.5 text-[13px] text-muted-foreground">
          {hint}
        </p>
      )}
      {error && (
        <p
          id={`${id}-error`}
          className="mt-1.5 flex items-start gap-2 text-[15px] text-danger"
          role="alert"
        >
          <WarningIcon size={18} className="mt-[3px] shrink-0" />
          {error}
        </p>
      )}
    </>
  );
}

const errorClass =
  "border-2 border-danger focus-visible:border-danger focus-visible:ring-danger/20";

function describedBy(id: string, hint?: string, error?: string): string | undefined {
  const ids = [hint && `${id}-hint`, error && `${id}-error`].filter(Boolean);
  return ids.length ? ids.join(" ") : undefined;
}

export function TextField({ label, hint, error, className, ...props }: FieldProps) {
  const id = useId();
  return (
    <div className={className}>
      <Label htmlFor={id}>{label}</Label>
      <Input
        id={id}
        type="text"
        aria-invalid={error ? true : undefined}
        aria-describedby={describedBy(id, hint, error)}
        className={cn(error && errorClass)}
        {...props}
      />
      <Messages id={id} hint={hint} error={error} />
    </div>
  );
}

/**
 * A passphrase field with the eye toggle. It starts hidden each time it
 * mounts and the choice is never stored; each field toggles on its own.
 */
export function PassphraseField({ label, hint, error, className, ...props }: FieldProps) {
  const id = useId();
  const [shown, setShown] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  const selection = useRef<[number | null, number | null] | null>(null);

  // Changing the type can move the caret to the end; put it back.
  // biome-ignore lint/correctness/useExhaustiveDependencies: runs after each flip of shown
  useLayoutEffect(() => {
    const el = input.current;
    const sel = selection.current;
    if (el && sel && document.activeElement === el) el.setSelectionRange(sel[0], sel[1]);
    selection.current = null;
  }, [shown]);

  return (
    <div className={className}>
      <Label htmlFor={id}>{label}</Label>
      <div className="relative">
        <Input
          ref={input}
          id={id}
          type={shown ? "text" : "password"}
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy(id, hint, error)}
          className={cn("pr-14", error && errorClass)}
          {...props}
        />
        <button
          type="button"
          aria-label={shown ? "Hide passphrase" : "Show passphrase"}
          aria-pressed={shown}
          aria-controls={id}
          // Keeps focus, and so the caret, in the field when tapped.
          onPointerDown={(e) => e.preventDefault()}
          onClick={() => {
            const el = input.current;
            if (el) selection.current = [el.selectionStart, el.selectionEnd];
            setShown((v) => !v);
          }}
          className="absolute top-1/2 right-1 grid size-11 -translate-y-1/2 place-items-center rounded-md text-muted-foreground hover:text-ink"
        >
          {shown ? <HideIcon size={20} /> : <ShowIcon size={20} />}
        </button>
      </div>
      <Messages id={id} hint={hint} error={error} />
    </div>
  );
}
