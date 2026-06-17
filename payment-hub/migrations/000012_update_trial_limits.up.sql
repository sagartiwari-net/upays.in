-- Free trial: 20 QR orders, 7 days (updated from 100/14)

UPDATE subscription_plans
SET
    validity_days = 7,
    order_limit = 20,
    features_json = '[{"text":"20 QR requests","included":true},{"text":"7 days","included":true},{"text":"0% transaction fee","included":true}]'
WHERE slug = 'trial';

-- Cap active trial subscriptions to new limits
UPDATE merchant_subscriptions ms
JOIN subscription_plans sp ON sp.id = ms.plan_id
SET
    ms.order_limit = 20,
    ms.expires_at = LEAST(ms.expires_at, DATE_ADD(ms.starts_at, INTERVAL 7 DAY))
WHERE sp.slug = 'trial' AND ms.status = 'active';
