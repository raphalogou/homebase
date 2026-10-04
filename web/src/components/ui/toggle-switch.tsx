import { Switch } from "@base-ui/react/switch";

// DESIGN.md "Toggle": 52 by 32 px, ink track when on.
export function ToggleSwitch({
  checked,
  onChange,
  label,
  disabled,
}: {
  checked: boolean;
  onChange: (on: boolean) => void;
  label: string;
  disabled?: boolean;
}) {
  return (
    <Switch.Root
      checked={checked}
      onCheckedChange={(on) => onChange(on)}
      aria-label={label}
      disabled={disabled ?? false}
      className="relative inline-flex h-8 w-[52px] shrink-0 items-center rounded-full border border-border-strong bg-soft transition-colors duration-150 data-[checked]:border-ink data-[checked]:bg-ink disabled:opacity-50"
    >
      <Switch.Thumb className="block size-6 translate-x-[3px] rounded-full bg-field shadow-segment transition-transform duration-150 data-[checked]:translate-x-[23px]" />
    </Switch.Root>
  );
}
