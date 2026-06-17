# Payment Links & Embed

Create shareable UPI payment links without coding — from the merchant dashboard or API.

## Dashboard (no code)

1. Go to **https://upays.in/dashboard/links**
2. Enter amount + product name
3. Click **Create link**
4. Share the payment URL (WhatsApp, email, SMS)

Each link creates a one-time UPI checkout session. Links expire in ~30 minutes if unpaid.

## Embed button on your site

After creating a link, use the payment URL in a button:

```html
<a href="https://upays.in/pay/YOUR_TOKEN" class="upipays-pay-btn">
  Pay with UPI
</a>
<link rel="stylesheet" href="https://upays.in/assets/css/upipays-button.css">
```

Or use the hosted style:

```html
<a href="PAYMENT_URL" style="display:inline-block;padding:12px 24px;background:#6d28d9;color:#fff;border-radius:8px;text-decoration:none;font-weight:600;">
  Pay ₹499 via UPI
</a>
```

## Server-side (PHP)

Use the PHP SDK to create links dynamically:

```php
require_once 'UpiPaysClient.php';
$client = new UpiPays_Client('https://upays.in', $apiKey, $apiSecret);
$result = $client->createOrder([
    'order_id'   => 'LINK-' . time(),
    'amount'     => 499.00,
    'currency'   => 'INR',
    'customer'   => ['email' => 'customer@example.com', 'name' => 'Customer'],
    'product'    => ['name' => 'Consultation fee'],
    'return_url' => 'https://yoursite.com/thank-you',
    'webhook_url'=> 'https://yoursite.com/upipays-webhook.php',
]);
header('Location: ' . $result['data']['payment_url']);
```

See `plugins/php-sdk/UpiPaysClient.php` and `plugins/payment-links/example-create-link.php`.

## Merchant API

Authenticated endpoint for dashboard / custom apps:

```
POST /merchant/api/payment-links
Authorization: Bearer {merchant_jwt}

{
  "amount": 499,
  "product_name": "Consultation",
  "return_url": "https://yoursite.com/done"
}
```

Response:

```json
{
  "order_id": "LINK-20260606120000",
  "payment_url": "https://upays.in/pay/...",
  "expires_at": "2026-06-06T12:30:00Z"
}
```
