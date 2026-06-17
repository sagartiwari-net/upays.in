-- UPIPays fresh database install
-- Run: mysql -u upipays -p upipays < migrations/install_upipays_full.sql

CREATE TABLE IF NOT EXISTS merchants (
    id              CHAR(36) PRIMARY KEY,
    name            VARCHAR(100) NOT NULL,
    domain          VARCHAR(255) NOT NULL UNIQUE,
    api_key         VARCHAR(64) NOT NULL UNIQUE,
    api_secret      VARCHAR(255) NOT NULL,
    webhook_url     VARCHAR(500) NOT NULL,
    return_url      VARCHAR(500) NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    payment_profile_id CHAR(36) NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_merchants_profile (payment_profile_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS admin_users (
    id              CHAR(36) PRIMARY KEY,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    name            VARCHAR(100) NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'admin',
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

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
    imap_last_ok_at DATETIME(3) NULL,
    imap_last_error VARCHAR(500) NULL,
    imap_last_checked_at DATETIME(3) NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS orders (
    id                  CHAR(36) PRIMARY KEY,
    hub_order_id        VARCHAR(20) NOT NULL UNIQUE,
    merchant_id         CHAR(36) NOT NULL,
    merchant_order_id   VARCHAR(100) NOT NULL,
    payment_token       VARCHAR(64) NOT NULL UNIQUE,
    amount              DECIMAL(12,2) NOT NULL,
    pay_amount          DECIMAL(12,2) NOT NULL,
    currency            VARCHAR(3) NOT NULL DEFAULT 'INR',
    payment_provider    VARCHAR(20) NOT NULL DEFAULT 'upi_email',
    payment_profile_id  CHAR(36) NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',
    customer_email      VARCHAR(255) NULL,
    customer_name       VARCHAR(255) NULL,
    customer_phone      VARCHAR(20) NULL,
    product_name        VARCHAR(255) NULL,
    product_description TEXT NULL,
    return_url          VARCHAR(500) NOT NULL,
    webhook_url         VARCHAR(500) NULL,
    phonepe_txn_id      VARCHAR(100) NULL,
    customer_utr        VARCHAR(32) NULL,
    phonepe_response    JSON NULL,
    paid_at             DATETIME(3) NULL,
    expires_at          DATETIME(3) NOT NULL,
    created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    UNIQUE KEY uk_merchant_order (merchant_id, merchant_order_id),
    KEY idx_orders_merchant_status (merchant_id, status, created_at),
    KEY idx_orders_payment_token (payment_token),
    KEY idx_orders_status_expires (status, expires_at),
    KEY idx_orders_pending_pay_amount (status, pay_amount),
    KEY idx_orders_pending_utr (status, customer_utr),
    KEY idx_orders_profile_status (payment_profile_id, status),
    CONSTRAINT fk_orders_merchant FOREIGN KEY (merchant_id) REFERENCES merchants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS processed_bank_txns (
    id              CHAR(36) PRIMARY KEY,
    utr             VARCHAR(32) NOT NULL UNIQUE,
    email_message_id VARCHAR(255) NOT NULL,
    amount          DECIMAL(12,2) NOT NULL,
    order_id        CHAR(36) NULL,
    payment_profile_id CHAR(36) NULL,
    raw_excerpt     TEXT NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_processed_order (order_id),
    KEY idx_processed_profile (payment_profile_id),
    CONSTRAINT fk_processed_order FOREIGN KEY (order_id) REFERENCES orders(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS webhook_logs (
    id              CHAR(36) PRIMARY KEY,
    order_id        CHAR(36) NOT NULL,
    merchant_id     CHAR(36) NOT NULL,
    direction       VARCHAR(10) NOT NULL,
    payload         JSON NOT NULL,
    response_code   INT NULL,
    response_body   TEXT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    retry_count     INT NOT NULL DEFAULT 0,
    next_retry_at   DATETIME(3) NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    KEY idx_webhook_logs_order (order_id),
    KEY idx_webhook_logs_retry (status, next_retry_at),
    CONSTRAINT fk_webhook_logs_order FOREIGN KEY (order_id) REFERENCES orders(id),
    CONSTRAINT fk_webhook_logs_merchant FOREIGN KEY (merchant_id) REFERENCES merchants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS refunds (
    id                  CHAR(36) PRIMARY KEY,
    order_id            CHAR(36) NOT NULL,
    merchant_id         CHAR(36) NOT NULL,
    amount              DECIMAL(12,2) NOT NULL,
    reason              TEXT NULL,
    phonepe_refund_id   VARCHAR(100) NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',
    initiated_by        CHAR(36) NULL,
    created_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at          DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_refunds_order FOREIGN KEY (order_id) REFERENCES orders(id),
    CONSTRAINT fk_refunds_merchant FOREIGN KEY (merchant_id) REFERENCES merchants(id),
    CONSTRAINT fk_refunds_admin FOREIGN KEY (initiated_by) REFERENCES admin_users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
