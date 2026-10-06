import { type FormEvent, useState } from "react";
import { ApiError, NetworkError } from "@/data/api";
import { useSettingsActions } from "@/data/settings";
import type { AccountInfo } from "@/data/types";
import {
  confirmError,
  PASSPHRASE_HINT,
  passphraseError,
  USERNAME_HINT,
  usernameError,
} from "@/lib/account";
import { useFieldErrors } from "@/lib/use-field-errors";
import { PassphraseField, TextField } from "./form-field";
import { Button } from "./ui/button";
import { DialogActions } from "./ui/dialog";

interface ChangeProps {
  /** Closes the form, after Cancel or a save. */
  onClose: () => void;
  onSaved: (info: AccountInfo) => void;
}

interface PassphraseProps extends ChangeProps {
  /** Replacing the owner's one-time passphrase: other labels, and the text
   * button logs out instead of cancelling. */
  first?: { onLogOut: () => void };
}

const OFFLINE = "This needs a connection. Try again when you are online.";

/** Puts a server refusal under the field it names, or under fallback. */
function serverMessage<F extends string>(
  err: unknown,
  fields: readonly F[],
  fallback: F,
  show: (f: F, m: string) => void,
) {
  if (err instanceof NetworkError) return show(fallback, OFFLINE);
  if (!(err instanceof ApiError)) throw err;
  const f = fields.find((x) => x === err.field) ?? fallback;
  show(f, err.message);
}

// Views of the Settings modal. Each form mounts when its view opens, so
// fields start empty and hidden each time.

export function UsernameForm({ onClose, onSaved }: ChangeProps) {
  const actions = useSettingsActions();
  const [username, setUsername] = useState("");
  const [passphrase, setPassphrase] = useState("");
  const [busy, setBusy] = useState(false);
  const errors = useFieldErrors<"username" | "passphrase">();
  const local = {
    username: usernameError(username),
    passphrase: passphrase ? "" : "Enter your passphrase.",
  };

  async function submit(e: FormEvent) {
    e.preventDefault();
    errors.submitted();
    if (local.username || local.passphrase) return;
    setBusy(true);
    try {
      onSaved(await actions.changeUsername(username, passphrase));
      onClose();
    } catch (err) {
      setBusy(false);
      serverMessage(err, ["username", "passphrase"] as const, "passphrase", errors.fromServer);
    }
  }

  return (
    <form onSubmit={submit} noValidate className="flex flex-col gap-5">
      <TextField
        label="New username"
        hint={`${USERNAME_HINT}.`}
        autoComplete="username"
        autoCapitalize="none"
        autoCorrect="off"
        spellCheck={false}
        value={username}
        autoFocus
        onChange={(e) => {
          setUsername(e.target.value);
          errors.edited("username");
        }}
        onBlur={() => errors.blur("username")}
        error={errors.error("username", local.username)}
      />
      <PassphraseField
        label="Your passphrase"
        autoComplete="current-password"
        value={passphrase}
        onChange={(e) => {
          setPassphrase(e.target.value);
          errors.edited("passphrase");
        }}
        onBlur={() => errors.blur("passphrase")}
        error={errors.error("passphrase", local.passphrase)}
      />
      <DialogActions>
        <Button type="submit" disabled={busy}>
          {busy ? "Saving" : "Save username"}
        </Button>
        <Button variant="text" className="self-center" onClick={onClose}>
          Cancel
        </Button>
      </DialogActions>
    </form>
  );
}

type PassField = "current" | "next" | "confirm";

export function PassphraseForm({ onClose, onSaved, first }: PassphraseProps) {
  const actions = useSettingsActions();
  const [values, setValues] = useState<Record<PassField, string>>({
    current: "",
    next: "",
    confirm: "",
  });
  const [busy, setBusy] = useState(false);
  const errors = useFieldErrors<PassField>();
  const local: Record<PassField, string> = {
    current: values.current
      ? ""
      : first
        ? "Enter the one-time passphrase."
        : "Enter your current passphrase.",
    next: passphraseError(values.next),
    confirm: confirmError(values.next, values.confirm),
  };

  async function submit(e: FormEvent) {
    e.preventDefault();
    errors.submitted();
    if (local.current || local.next || local.confirm) return;
    setBusy(true);
    try {
      onSaved(await actions.changePassphrase(values.current, values.next));
      onClose();
    } catch (err) {
      setBusy(false);
      serverMessage(err, ["current", "next"] as const, "current", errors.fromServer);
    }
  }

  const field = (f: PassField) => ({
    value: values[f],
    onChange: (e: { target: { value: string } }) => {
      setValues((v) => ({ ...v, [f]: e.target.value }));
      errors.edited(f);
    },
    onBlur: () => errors.blur(f),
    error: errors.error(f, local[f]),
  });

  return (
    <form onSubmit={submit} noValidate className="flex flex-col gap-5">
      <PassphraseField
        label={first ? "One-time passphrase" : "Current passphrase"}
        autoComplete="current-password"
        autoFocus
        {...field("current")}
      />
      <PassphraseField
        label="New passphrase"
        hint={PASSPHRASE_HINT}
        autoComplete="new-password"
        {...field("next")}
      />
      <PassphraseField
        label="Confirm new passphrase"
        autoComplete="new-password"
        {...field("confirm")}
      />
      <DialogActions>
        <Button type="submit" disabled={busy}>
          {busy ? "Saving" : "Save passphrase"}
        </Button>
        <Button variant="text" className="self-center" onClick={first?.onLogOut ?? onClose}>
          {first ? "Log out" : "Cancel"}
        </Button>
      </DialogActions>
    </form>
  );
}
