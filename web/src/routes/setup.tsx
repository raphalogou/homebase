import { type FormEvent, useState } from "react";
import { AuthPage } from "@/components/auth-page";
import { PassphraseField, TextField } from "@/components/form-field";
import { Button } from "@/components/ui/button";
import { ApiError, NetworkError } from "@/data/api";
import { useAuth } from "@/data/hooks";
import {
  confirmError,
  PASSPHRASE_HINT,
  passphraseError,
  USERNAME_HINT,
  usernameError,
} from "@/lib/account";
import { useFieldErrors } from "@/lib/use-field-errors";

type Field = "username" | "passphrase" | "confirm";

// Shown on a fresh install until the account exists; then it logs in and
// the router moves on to Today.
export default function Setup() {
  const { setup } = useAuth();
  const [username, setUsername] = useState("");
  const [passphrase, setPassphrase] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const errors = useFieldErrors<Field>();

  const local: Record<Field, string> = {
    username: usernameError(username),
    passphrase: passphraseError(passphrase),
    confirm: confirmError(passphrase, confirm),
  };

  async function submit(e: FormEvent) {
    e.preventDefault();
    errors.submitted();
    if (local.username || local.passphrase || local.confirm) return;
    setBusy(true);
    try {
      await setup(username, passphrase);
    } catch (err) {
      setBusy(false);
      if (err instanceof NetworkError)
        errors.fromServer("confirm", "Cannot reach Homebase. Check your connection and try again.");
      else if (err instanceof ApiError)
        errors.fromServer(
          err.field === "username"
            ? "username"
            : err.field === "passphrase"
              ? "passphrase"
              : "confirm",
          err.message,
        );
      else throw err;
    }
  }

  const field = (f: Field, set: (v: string) => void) => ({
    onChange: (e: { target: { value: string } }) => {
      set(e.target.value);
      errors.edited(f);
    },
    onBlur: () => errors.blur(f),
    error: errors.error(f, local[f]),
  });

  return (
    <AuthPage title="Set up Homebase" statement={"One account, just\nfor you."}>
      <form onSubmit={submit} className="mt-5 flex flex-col gap-5" noValidate>
        <TextField
          label="Username"
          placeholder={USERNAME_HINT}
          autoComplete="username"
          autoCapitalize="none"
          autoCorrect="off"
          spellCheck={false}
          value={username}
          autoFocus
          {...field("username", setUsername)}
        />
        <PassphraseField
          label="Passphrase"
          hint={PASSPHRASE_HINT}
          autoComplete="new-password"
          value={passphrase}
          {...field("passphrase", setPassphrase)}
        />
        <PassphraseField
          label="Confirm passphrase"
          autoComplete="new-password"
          value={confirm}
          {...field("confirm", setConfirm)}
        />
        <Button type="submit" className="mt-1 w-full" disabled={busy}>
          {busy ? "Creating account" : "Create account"}
        </Button>
      </form>
    </AuthPage>
  );
}
