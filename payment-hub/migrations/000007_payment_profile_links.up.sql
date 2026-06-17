ALTER TABLE merchants
    ADD COLUMN payment_profile_id CHAR(36) NULL AFTER status,
    ADD KEY idx_merchants_profile (payment_profile_id);

ALTER TABLE orders
    ADD COLUMN payment_profile_id CHAR(36) NULL AFTER payment_provider,
    ADD KEY idx_orders_profile_status (payment_profile_id, status);

ALTER TABLE processed_bank_txns
    ADD COLUMN payment_profile_id CHAR(36) NULL AFTER order_id,
    ADD KEY idx_processed_profile (payment_profile_id);
