-- Daily backups by the server itself, switched on in Settings. Off until chosen.
ALTER TABLE settings ADD COLUMN backups INTEGER NOT NULL DEFAULT 0 CHECK (backups IN (0, 1));
