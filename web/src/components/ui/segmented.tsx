import { Toggle } from "@base-ui/react/toggle";
import { ToggleGroup } from "@base-ui/react/toggle-group";
import { cn } from "@/lib/utils";

interface SegmentedProps<T extends string> {
  label: string;
  value: T;
  options: { value: T; label: string }[];
  onChange: (value: T) => void;
  className?: string;
}

// DESIGN.md: two or three buttons in a --soft track, radius 12 (track) and
// 9 (active segment); the active segment is lifted with the small shadow.
export function Segmented<T extends string>({
  label,
  value,
  options,
  onChange,
  className,
}: SegmentedProps<T>) {
  return (
    <ToggleGroup
      aria-label={label}
      value={[value]}
      onValueChange={(next) => {
        const v = next[0] as T | undefined;
        if (v !== undefined) onChange(v);
      }}
      className={cn("flex rounded-lg bg-soft p-1", className)}
    >
      {options.map((o) => (
        <Toggle
          key={o.value}
          value={o.value}
          className="h-11 flex-1 rounded-[9px] px-3 text-sm font-semibold text-muted-foreground data-[pressed]:bg-field data-[pressed]:text-ink data-[pressed]:shadow-[0_1px_2px_rgba(20,25,24,.14)]"
        >
          {o.label}
        </Toggle>
      ))}
    </ToggleGroup>
  );
}
