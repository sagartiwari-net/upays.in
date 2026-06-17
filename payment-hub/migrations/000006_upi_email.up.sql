ALTER TABLE orders
    ADD COLUMN pay_amount DECIMAL(12,2) NULL AFTER amount,
    ADD COLUMN payment_provider VARCHAR(20) NOT NULL DEFAULT 'upi_email' AFTER currency,
    ADD COLUMN customer_utr VARCHAR(32) NULL AFTER phonepe_txn_id;

UPDATE orders SET pay_amount = amount WHERE pay_amount IS NULL;

ALTER TABLE orders MODIFY pay_amount DECIMAL(12,2) NOT NULL;

CREATE INDEX idx_orders_pending_pay_amount ON orders(status, pay_amount);
CREATE INDEX idx_orders_pending_utr ON orders(status, customer_utr);

CREATE TABLE IF NOT EXISTS payment_profiles (
    id              CHAR(36) PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    upi_id          VARCHAR(100) NOT NULL,
    payee_name      VARCHAR(100) NOT NULL DEFAULT 'UPIPays',
    bank_code       VARCHAR(20) NOT NULL DEFAULT 'hdfc',
    imap_host       VARCHAR(255) NOT NULL,
    imap_port       INT NOT NULL DEFAULT 993,
    imap_user       VARCHAR(255) NOT NULL,
    imap_password   VARCHAR(255) NOT NULL,
    sender_filter   VARCHAR(255) NOT NULL,
    parser_type     VARCHAR(20) NOT NULL DEFAULT 'hdfc',
    is_active       TINYINT(1) NOT NULL DEFAULT 1,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS processed_bank_txns (
    id              CHAR(36) PRIMARY KEY,
    utr             VARCHAR(32) NOT NULL UNIQUE,
    email_message_id VARCHAR(255) NOT NULL,
    amount          DECIMAL(12,2) NOT NULL,
    order_id        CHAR(36) NULL,
    raw_excerpt     TEXT NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_processed_order (order_id),
    CONSTRAINT fk_processed_order FOREIGN KEY (order_id) REFERENCES orders(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
