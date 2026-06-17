ALTER TABLE processed_bank_txns DROP COLUMN payment_profile_id;
ALTER TABLE orders DROP INDEX idx_orders_profile_status, DROP COLUMN payment_profile_id;
ALTER TABLE merchants DROP INDEX idx_merchants_profile, DROP COLUMN payment_profile_id;
