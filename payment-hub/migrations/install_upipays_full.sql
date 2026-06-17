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

CREATE TABLE IF NOT EXISTS merchant_users (
    id              CHAR(36) PRIMARY KEY,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    merchant_id     CHAR(36) NOT NULL UNIQUE,
    onboarding_done TINYINT(1) NOT NULL DEFAULT 0,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    CONSTRAINT fk_merchant_users_merchant FOREIGN KEY (merchant_id) REFERENCES merchants(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS subscription_plans (
    id              CHAR(36) PRIMARY KEY,
    slug            VARCHAR(50) NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    price_inr       DECIMAL(10,2) NOT NULL DEFAULT 0,
    validity_days   INT NOT NULL DEFAULT 28,
    order_limit     INT NOT NULL,
    is_recommended  TINYINT(1) NOT NULL DEFAULT 0,
    sort_order      INT NOT NULL DEFAULT 0,
    is_active       TINYINT(1) NOT NULL DEFAULT 1,
    features_json   JSON NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS merchant_subscriptions (
    id              CHAR(36) PRIMARY KEY,
    merchant_id     CHAR(36) NOT NULL,
    plan_id         CHAR(36) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    starts_at       DATETIME(3) NOT NULL,
    expires_at      DATETIME(3) NOT NULL,
    orders_used     INT NOT NULL DEFAULT 0,
    order_limit     INT NOT NULL,
    activated_by    VARCHAR(100) NULL,
    notes           VARCHAR(500) NULL,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_msub_merchant (merchant_id, status),
    KEY idx_msub_expires (expires_at),
    CONSTRAINT fk_msub_merchant FOREIGN KEY (merchant_id) REFERENCES merchants(id) ON DELETE CASCADE,
    CONSTRAINT fk_msub_plan FOREIGN KEY (plan_id) REFERENCES subscription_plans(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO subscription_plans (id, slug, name, price_inr, validity_days, order_limit, is_recommended, sort_order, features_json) VALUES
('plan-trial-001', 'trial', 'Free Trial', 0, 7, 20, 0, 0,
 '[{"text":"20 QR requests","included":true},{"text":"7 days","included":true},{"text":"0% transaction fee","included":true}]'),
('plan-starter-001', 'starter', 'Starter', 499, 28, 5000, 0, 1,
 '[{"text":"5,000 QR requests","included":true},{"text":"0% transaction fee","included":true},{"text":"Webhook callbacks","included":true}]'),
('plan-growth-001', 'growth', 'Growth', 999, 28, 10000, 0, 2,
 '[{"text":"10,000 QR requests","included":true},{"text":"0% transaction fee","included":true},{"text":"Webhook callbacks","included":true}]'),
('plan-business-001', 'business', 'Business', 1999, 28, 15000, 1, 3,
 '[{"text":"15,000 QR requests","included":true},{"text":"Priority support","included":true},{"text":"0% transaction fee","included":true}]'),
('plan-pro-001', 'pro', 'Pro', 4999, 28, 25000, 0, 4,
 '[{"text":"25,000 QR requests","included":true},{"text":"Multiple UPI profiles","included":true},{"text":"0% transaction fee","included":true}]')
ON DUPLICATE KEY UPDATE name = VALUES(name);

CREATE TABLE IF NOT EXISTS cms_pages (
    id              CHAR(36) PRIMARY KEY,
    slug            VARCHAR(100) NOT NULL UNIQUE,
    title           VARCHAR(200) NOT NULL,
    meta_description VARCHAR(500) NULL,
    body_html       MEDIUMTEXT NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'draft',
    show_in_nav     TINYINT(1) NOT NULL DEFAULT 0,
    nav_label       VARCHAR(100) NULL,
    sort_order      INT NOT NULL DEFAULT 0,
    created_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    updated_at      DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    KEY idx_cms_status (status),
    KEY idx_cms_nav (show_in_nav, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
