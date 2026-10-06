-- Several people, each with their own database (data/users/<id>/). This
-- file's account row is that person. The owner adds and removes the others;
-- an added person starts with a one-time passphrase they must change.
ALTER TABLE account ADD COLUMN owner INTEGER NOT NULL DEFAULT 0 CHECK (owner IN (0, 1));
ALTER TABLE account ADD COLUMN must_change INTEGER NOT NULL DEFAULT 0 CHECK (must_change IN (0, 1));
-- Before this, the account was the only one, so it is the owner.
UPDATE account SET owner = 1;
