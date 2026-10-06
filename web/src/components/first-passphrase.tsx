import { useAuth, useVersion } from "@/data/hooks";
import { useData } from "@/data/provider";
import { PassphraseForm } from "./account-forms";
import { CheckIcon } from "./icons";
import { useNotify } from "./toaster";
import { ResponsiveDialog } from "./ui/dialog";

// Someone the owner added logs in with a one-time passphrase. Until they
// choose their own, this covers the app and cannot be dismissed; the only
// other way out is logging out.
export function FirstPassphrase() {
  const { store } = useData();
  const { logout } = useAuth();
  const notify = useNotify();
  useVersion();
  const open = store.me?.mustChange === true;
  return (
    <ResponsiveDialog
      open={open}
      onOpenChange={() => {}}
      title="Choose your passphrase"
      description="Your account was made with a one-time passphrase. Choose your own to start using Homebase."
    >
      {open && (
        <PassphraseForm
          onClose={() => {}}
          onSaved={() => notify({ title: "Passphrase saved", icon: CheckIcon })}
          first={{ onLogOut: () => void logout() }}
        />
      )}
    </ResponsiveDialog>
  );
}
