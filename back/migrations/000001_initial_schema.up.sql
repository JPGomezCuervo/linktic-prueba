CREATE TABLE IF NOT EXISTS items(
	id TEXT NOT NULL PRIMARY KEY,
	name TEXT NOT NULL,
	units INTEGER NOT NULL,
	price INTEGER NOT NULL,
	deleted INTEGER DEFAULT 0,
	created_at INTEGER DEFAULT (unixepoch()),
	updated_at INTEGER DEFAULT (unixepoch())
) STRICT;

CREATE TRIGGER update_items_timestamp
AFTER UPDATE ON items
BEGIN
   UPDATE items SET updated_at = unixepoch() WHERE id = NEW.id;
END;

CREATE TABLE IF NOT EXISTS accounts(
	id TEXT NOT NULL PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	deleted INTEGER NOT NULL DEFAULT 0 CHECK (deleted IN (0, 1)),
	created_at INTEGER DEFAULT (unixepoch()),
	updated_at INTEGER DEFAULT (unixepoch())
) STRICT;

CREATE TRIGGER update_accounts_timestamp
AFTER UPDATE ON accounts
BEGIN
   UPDATE accounts SET updated_at = unixepoch() WHERE id = NEW.id;
END;
