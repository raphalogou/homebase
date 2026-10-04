import { useEffect, useId, useRef, useState } from "react";
import { Label, Textarea } from "./ui/input";

const PAUSE_MS = 800;

// Notes save when typing pauses and when the field loses focus.
export function NotesField({
  value,
  onSave,
  label = "Notes",
}: {
  value: string;
  onSave: (v: string) => Promise<void> | void;
  label?: string;
}) {
  const [text, setText] = useState(value);
  const [saved, setSaved] = useState(true);
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const latest = useRef(value);
  const id = useId();

  // Another device may change the notes; take them when not mid-edit.
  useEffect(() => {
    if (saved) setText(value);
    latest.current = value;
  }, [value, saved]);

  useEffect(() => () => clearTimeout(timer.current), []);

  function save(v: string) {
    clearTimeout(timer.current);
    if (v !== latest.current) {
      latest.current = v;
      void onSave(v);
    }
    setSaved(true);
  }

  return (
    <div>
      <Label htmlFor={id}>{label}</Label>
      <Textarea
        id={id}
        value={text}
        maxLength={20000}
        onChange={(e) => {
          const v = e.target.value;
          setText(v);
          setSaved(false);
          clearTimeout(timer.current);
          timer.current = setTimeout(() => save(v), PAUSE_MS);
        }}
        onBlur={() => save(text)}
      />
    </div>
  );
}
