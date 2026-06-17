-- Subscription plans and merchant subscriptions (Phase 3)

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

-- Default plans (match marketing pricing page)
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

-- Backfill trial for merchants created before subscriptions
INSERT INTO merchant_subscriptions (id, merchant_id, plan_id, status, starts_at, expires_at, orders_used, order_limit, notes)
SELECT
    LOWER(CONCAT(
        SUBSTRING(MD5(CONCAT(m.id, '-trial')), 1, 8), '-',
        SUBSTRING(MD5(CONCAT(m.id, '-trial')), 9, 4), '-',
        SUBSTRING(MD5(CONCAT(m.id, '-trial')), 13, 4), '-',
        SUBSTRING(MD5(CONCAT(m.id, '-trial')), 17, 4), '-',
        SUBSTRING(MD5(CONCAT(m.id, '-trial')), 21, 12)
    )),
    m.id,
    'plan-trial-001',
    'active',
    UTC_TIMESTAMP(3),
    DATE_ADD(UTC_TIMESTAMP(3), INTERVAL 7 DAY),
    COALESCE((SELECT COUNT(*) FROM orders o WHERE o.merchant_id = m.id), 0),
    20,
    'backfill trial'
FROM merchants m
WHERE NOT EXISTS (
    SELECT 1 FROM merchant_subscriptions ms WHERE ms.merchant_id = m.id
);
