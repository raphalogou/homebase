import { useEffect, useId, useState } from "react";
import { cn } from "@/lib/utils";

/** A heading you can edit in place; saved on Enter or when focus leaves. */
export function TitleField({
  value,
  label,
  onSave,
  className,
}: {
  value: string;
  label: string;
  onSave: (v: string) => void;
  className?: string;
}) {
  const [text, setText] = useState(value);
  const id = useId();
  useEffect(() => setText(value), [value]);
  const save = () => {
    const clean = text.trim();
    if (!clean) setText(value);
    else if (clean !== value) onSave(clean.slice(0, 300));
  };
  return (
    <>
      <label htmlFor={id} className="sr-only">
        {label}
      </label>
      <textarea
        id={id}
        rows={1}
        value={text}
        maxLength={300}
        onChange={(e) => setText(e.target.value.replace(/\n/g, " "))}
        onBlur={save}
        onKeyDown={(e) => {
          if (e.key === "Enter") {
            e.preventDefault();
            e.currentTarget.blur();
          }
        }}
        className={cn(
          "-mx-2 block w-[calc(100%+16px)] resize-none overflow-hidden rounded-md bg-transparent px-2 [field-sizing:content] hover:bg-soft/60",
          className,
        )}
      />
    </>
  );
}
