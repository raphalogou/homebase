import { cn } from "@/lib/utils";
import { CheckIcon } from "./icons";

interface DoneBoxProps {
  done: boolean;
  onToggle: () => void;
  title: string;
}

// A real checkbox, drawn as a 26 px circle inside a 44 px target
// (DESIGN.md "Space and shape").
export function DoneBox({ done, onToggle, title }: DoneBoxProps) {
  return (
    <label className="-ml-[9px] grid size-11 shrink-0 cursor-pointer place-items-center rounded-full">
      <input
        type="checkbox"
        checked={done}
        onChange={onToggle}
        className="peer sr-only"
        aria-label={done ? `Mark not done: ${title}` : `Mark done: ${title}`}
      />
      <span
        aria-hidden="true"
        className={cn(
          "grid size-[26px] place-items-center rounded-full border-[1.8px] border-ink peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-ink",
          done && "bg-ink text-field",
        )}
      >
        {done && <CheckIcon size={18} />}
      </span>
    </label>
  );
}
