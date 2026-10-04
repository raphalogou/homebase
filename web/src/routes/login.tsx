import { type FormEvent, useState } from "react";
import { AuthPage } from "@/components/auth-page";
import { PassphraseField, TextField } from "@/components/form-field";
import { NoticeIcon } from "@/components/icons";
import { Button } from "@/components/ui/button";
import { ApiError, NetworkError } from "@/data/api";
import { useAuth } from "@/data/hooks";
import { useFieldErrors } from "@/lib/use-field-errors";

type Field = "username" | "passphrase";

export default function Login() {
  const { login, sessionEnded, lastUsername } = useAuth();
  const [username, setUsername] = useState(lastUsername);
  const [passphrase, setPassphrase] = useState("");
  const [busy, setBusy] = useState(false);
  const errors = useFieldErrors<Field>();

  const local: Record<Field, string> = {
    username: username.trim() ? "" : "Enter your username.",
    passphrase: passphrase ? "" : "Enter your passphrase.",
  };

  async function submit(e: FormEvent) {
    e.preventDefault();
    errors.submitted();
    if (local.username || local.passphrase) return;
    setBusy(true);
    try {
      await login(username, passphrase);
    } catch (err) {
      setBusy(false);
      // A refused pair is shown under the passphrase, never saying which part was wrong.
      if (err instanceof NetworkError)
        errors.fromServer(
          "passphrase",
          "Cannot reach Homebase. Check your connection and try again.",
        );
      else if (err instanceof ApiError)
        errors.fromServer(err.field === "username" ? "username" : "passphrase", err.message);
      else throw err;
    }
  }

  return (
    <AuthPage title="Log in" intro="Enter your username and passphrase.">
      {sessionEnded && (
        <div className="mt-6 flex gap-3 rounded-lg bg-soft px-4 py-3" role="status">
          <NoticeIcon size={18} className="mt-[3px] shrink-0" />
          <p>
            <strong className="block font-semibold">Your session ended</strong>
            <span className="text-muted-foreground">
              Log in again. Changes made offline are kept and will sync.
            </span>
          </p>
        </div>
      )}
      <form onSubmit={submit} className="mt-6 flex flex-col gap-5" noValidate>
        <TextField
          label="Username"
          autoComplete="username"
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
          value={username}
          onChange={(e) => {
            setUsername(e.target.value);
            errors.edited("username");
          }}
          onBlur={() => errors.blur("username")}
          error={errors.error("username", local.username)}
          autoFocus={!lastUsername}
        />
        <PassphraseField
          label="Passphrase"
          autoComplete="current-password"
          value={passphrase}
          onChange={(e) => {
            setPassphrase(e.target.value);
            errors.edited("passphrase");
          }}
          onBlur={() => errors.blur("passphrase")}
          error={errors.error("passphrase", local.passphrase)}
          autoFocus={!!lastUsername}
        />
        <Button type="submit" className="mt-1 w-full" disabled={busy}>
          {busy ? "Logging in" : "Log in"}
        </Button>
      </form>
    </AuthPage>
  );
}
