// The account rules from docs/SPEC.md section 7, checked in the forms so
// a mistake shows before the request. The server checks them again.

export const MIN_PASSPHRASE = 12;
export const MAX_PASSPHRASE = 1000;

export const PASSPHRASE_HINT = "At least 12 characters. Several random words work well.";
export const USERNAME_HINT = "Letters, numbers, dots and dashes";
export const USERNAME_ERROR = "Use 3 to 32 letters, numbers, dots or dashes.";
export const MISMATCH = "The two passphrases do not match.";

/** Usernames compare without case; the server stores them in lowercase. */
export function normalizeUsername(s: string): string {
  return s.trim().toLowerCase();
}

/** Why a username cannot be used, or "" when it can. */
export function usernameError(s: string): string {
  return /^[a-z0-9._-]{3,32}$/.test(normalizeUsername(s)) ? "" : USERNAME_ERROR;
}

/** Why a new passphrase cannot be used, or "". Counts characters, not UTF-16 units. */
export function passphraseError(s: string): string {
  const n = [...s].length;
  if (n < MIN_PASSPHRASE) return `Use at least ${MIN_PASSPHRASE} characters. You have ${n}.`;
  if (n > MAX_PASSPHRASE) return `Use at most ${MAX_PASSPHRASE} characters.`;
  return "";
}

export function confirmError(passphrase: string, confirm: string): string {
  return passphrase === confirm ? "" : MISMATCH;
}
