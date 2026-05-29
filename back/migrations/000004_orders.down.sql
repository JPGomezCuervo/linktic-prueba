DROP TRIGGER IF EXISTS update_orders_timestamp;
DROP INDEX IF EXISTS idx_orders_status_delivery;
DROP INDEX IF EXISTS idx_orders_account_created;
DROP TABLE IF EXISTS orders;
