import { type FormEvent, useId, useState } from "react";
import { Wordmark } from "@/components/app-shell";
import { ScreenTitle } from "@/components/section";
import { Button } from "@/components/ui/button";
import { Input, Label } from "@/components/ui/input";
import { ApiError, NetworkError } from "@/data/api";
import { useAuth } from "@/data/hooks";

// Approved design: the same top-aligned column as the other screens.
export default function Login() {
  const { login } = useAuth();
  const [passphrase, setPassphrase] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const id = useId();

  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!passphrase) {
      setMessage("Enter your passphrase.");
      return;
    }
    setBusy(true);
    setMessage("");
    try {
      await login(passphrase);
    } catch (err) {
      if (err instanceof NetworkError)
        setMessage("Cannot reach Homebase. Check your connection and try again.");
      else if (err instanceof ApiError) setMessage(err.message);
      else throw err;
      setBusy(false);
    }
  }

  return (
    <main className="mx-auto w-full max-w-[400px] px-6 pt-5 min-[900px]:pt-24">
      <Wordmark />
      <ScreenTitle className="mt-10">Log in</ScreenTitle>
      <p className="mt-2 text-muted-foreground">Enter your passphrase to open your planner.</p>
      <form onSubmit={submit} className="mt-8" noValidate>
        <Label htmlFor={id}>Passphrase</Label>
        <Input
          id={id}
          type="password"
          autoComplete="current-password"
          value={passphrase}
          onChange={(e) => setPassphrase(e.target.value)}
          aria-describedby={message ? `${id}-msg` : undefined}
          aria-invalid={message ? true : undefined}
          autoFocus
        />
        {message && (
          <p id={`${id}-msg`} className="mt-2 text-sm" role="alert">
            {message}
          </p>
        )}
        <Button type="submit" className="mt-6 w-full" disabled={busy}>
          Log in
        </Button>
      </form>
    </main>
  );
}
