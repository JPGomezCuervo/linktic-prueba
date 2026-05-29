CREATE TABLE IF NOT EXISTS orders(
	id TEXT NOT NULL PRIMARY KEY,
	account_id TEXT NOT NULL,
	item_id TEXT NOT NULL,
	units INTEGER NOT NULL CHECK (units > 0),
	unit_price INTEGER NOT NULL CHECK (unit_price >= 0),
	total_price INTEGER NOT NULL CHECK (total_price >= 0),
	payment_method TEXT NOT NULL CHECK (payment_method IN ('credit_card', 'checking_account')),
	status TEXT NOT NULL CHECK (status IN ('pending', 'completed')),
	delivery_at INTEGER NOT NULL,
	completed_at INTEGER,
	created_at INTEGER DEFAULT (unixepoch()),
	updated_at INTEGER DEFAULT (unixepoch()),
	FOREIGN KEY (account_id) REFERENCES accounts(id),
	FOREIGN KEY (item_id) REFERENCES items(id)
) STRICT;

CREATE INDEX IF NOT EXISTS idx_orders_account_created
ON orders(account_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_orders_status_delivery
ON orders(status, delivery_at);

CREATE TRIGGER IF NOT EXISTS update_orders_timestamp
AFTER UPDATE ON orders
BEGIN
   UPDATE orders SET updated_at = unixepoch() WHERE id = NEW.id;
END;
