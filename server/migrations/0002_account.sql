-- One account: a username and an argon2id passphrase hash. Before this, the
-- hash lived only in HOMEBASE_PASSPHRASE_HASH; the server copies it here on
-- first start, with the username "owner".
CREATE TABLE account (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  username TEXT NOT NULL CHECK (length(username) BETWEEN 3 AND 32 AND username = lower(username)),
  passphrase_hash TEXT NOT NULL,
  changed_at INTEGER NOT NULL                        -- when the passphrase was last set
);
