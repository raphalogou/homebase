import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { setTheme, type Theme, useTheme } from "@/lib/theme";
import { DarkIcon, LightIcon, SystemIcon } from "./icons";

const THEMES = [
  { value: "system", label: "System", Icon: SystemIcon },
  { value: "light", label: "Light", Icon: LightIcon },
  { value: "dark", label: "Dark", Icon: DarkIcon },
] as const;

/** The rail's theme menu: the same choice as Settings, a click closer. */
export function ThemeToggle() {
  const theme = useTheme();
  const current = THEMES.find((t) => t.value === theme) ?? THEMES[0];
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={`Theme: ${current.label}`}
        className="flex min-h-11 items-center gap-3 rounded-md px-3 text-[15px] font-medium hover:bg-soft data-popup-open:bg-soft"
      >
        <current.Icon size={20} />
        Theme
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top">
        <DropdownMenuRadioGroup
          value={theme}
          onValueChange={(v: Theme) => {
            setTheme(v);
          }}
        >
          {THEMES.map(({ value, label, Icon }) => (
            <DropdownMenuRadioItem key={value} value={value} closeOnClick>
              <Icon size={20} />
              {label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
