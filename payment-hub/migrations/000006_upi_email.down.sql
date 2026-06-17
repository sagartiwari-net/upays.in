DROP TABLE IF EXISTS processed_bank_txns;
DROP TABLE IF EXISTS payment_profiles;
ALTER TABLE orders
    DROP INDEX idx_orders_pending_utr,
    DROP INDEX idx_orders_pending_pay_amount,
    DROP COLUMN customer_utr,
    DROP COLUMN payment_provider,
    DROP COLUMN pay_amount;
