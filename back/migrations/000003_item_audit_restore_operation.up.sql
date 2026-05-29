CREATE TABLE IF NOT EXISTS item_audit_logs_new(
	id TEXT NOT NULL PRIMARY KEY,
	item_id TEXT NOT NULL,
	operation TEXT NOT NULL CHECK (operation IN ('create', 'update', 'delete', 'restore')),
	changes_json TEXT NOT NULL,
	actor_account_id TEXT NOT NULL,
	actor_name TEXT NOT NULL,
	actor_email TEXT NOT NULL,
	created_at INTEGER DEFAULT (unixepoch()),
	FOREIGN KEY (item_id) REFERENCES items(id),
	FOREIGN KEY (actor_account_id) REFERENCES accounts(id)
) STRICT;

INSERT INTO item_audit_logs_new(id, item_id, operation, changes_json, actor_account_id, actor_name, actor_email, created_at)
SELECT id, item_id, operation, changes_json, actor_account_id, actor_name, actor_email, created_at
FROM item_audit_logs;

DROP TABLE item_audit_logs;

ALTER TABLE item_audit_logs_new RENAME TO item_audit_logs;

CREATE INDEX IF NOT EXISTS idx_item_audit_logs_item_created
ON item_audit_logs(item_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_item_audit_logs_actor_created
ON item_audit_logs(actor_account_id, created_at DESC);
