<?php
/**
 * Example: create a UPIPays payment link and redirect customer.
 * Copy UpiPaysClient.php from plugins/php-sdk/ alongside this file.
 */
require_once __DIR__ . '/../php-sdk/UpiPaysClient.php';

$apiKey    = getenv('UPIPAYS_API_KEY') ?: 'mk_your_key';
$apiSecret = getenv('UPIPAYS_API_SECRET') ?: 'sk_your_secret';
$hubUrl    = 'https://upays.in';

$amount = isset($_GET['amount']) ? (float) $_GET['amount'] : 100.0;
$name   = isset($_GET['name']) ? (string) $_GET['name'] : 'Payment';

$client = new UpiPays_Client($hubUrl, $apiKey, $apiSecret);

try {
    $result = $client->createOrder([
        'order_id'    => 'LINK-' . time(),
        'amount'      => $amount,
        'currency'    => 'INR',
        'customer'    => ['email' => 'customer@example.com', 'name' => $name],
        'product'     => ['name' => $name],
        'return_url'  => 'https://yoursite.com/thank-you',
        'webhook_url' => 'https://yoursite.com/upipays-webhook.php',
    ]);
    header('Location: ' . $result['data']['payment_url']);
    exit;
} catch (Exception $e) {
    http_response_code(500);
    echo 'Payment link error: ' . htmlspecialchars($e->getMessage());
}
