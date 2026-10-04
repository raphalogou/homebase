import { type FormEvent, useId, useState } from "react";
import { ActionError } from "@/data/hooks";
import { cn } from "@/lib/utils";
import { Button } from "./ui/button";
import { Input } from "./ui/input";

interface AddFieldProps {
  label: string;
  placeholder: string;
  button: string;
  onAdd: (text: string) => Promise<unknown>;
  serif?: boolean;
  showLabel?: boolean;
}

/** A field and a button that adds what was typed, like the capture bar. */
export function AddField({ label, placeholder, button, onAdd, serif, showLabel }: AddFieldProps) {
  const [text, setText] = useState("");
  const [message, setMessage] = useState("");
  const id = useId();

  async function submit(e: FormEvent) {
    e.preventDefault();
    setMessage("");
    // Clear at once, so anything typed while saving is not wiped afterwards.
    const value = text;
    setText("");
    try {
      await onAdd(value);
    } catch (err) {
      setText((current) => current || value);
      if (err instanceof ActionError) setMessage(err.message);
      else throw err;
    }
  }

  return (
    <form onSubmit={submit}>
      <label
        htmlFor={id}
        className={showLabel ? "mb-2 block text-sm font-semibold text-muted-foreground" : "sr-only"}
      >
        {label}
      </label>
      <div className="flex gap-2">
        <Input
          id={id}
          value={text}
          maxLength={300}
          placeholder={placeholder}
          onChange={(e) => setText(e.target.value)}
          className={cn(serif && "font-serif text-[17px]")}
          autoComplete="off"
        />
        <Button type="submit" className="h-12 shrink-0 px-4 text-sm">
          {button}
        </Button>
      </div>
      {message && (
        <p className="mt-2 text-sm" role="status">
          {message}
        </p>
      )}
    </form>
  );
}
