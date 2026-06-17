<?php
defined('ABSPATH') || exit;

/**
 * Standalone UPIPays client (no wp_remote dependency for testing).
 */
class UpiPays_WC_Client
{
    private $hubUrl;
    private $apiKey;
    private $apiSecret;

    public function __construct($hubUrl, $apiKey, $apiSecret)
    {
        $this->hubUrl = rtrim((string) $hubUrl, '/');
        $this->apiKey = (string) $apiKey;
        $this->apiSecret = (string) $apiSecret;
    }

    public function sign($timestamp, $method, $path, $body = '')
    {
        $message = $timestamp . '|' . strtoupper($method) . '|' . $path . '|' . $body;
        return hash_hmac('sha256', $message, $this->apiSecret);
    }

    public function request($method, $path, $payload = null)
    {
        $body = '';
        if ($payload !== null) {
            $body = wp_json_encode($payload);
        }
        $timestamp = (string) time();
        $signature = $this->sign($timestamp, $method, $path, $body);

        $args = array(
            'method'  => strtoupper($method),
            'timeout' => 30,
            'headers' => array(
                'Content-Type'   => 'application/json',
                'X-Merchant-Key' => $this->apiKey,
                'X-Timestamp'    => $timestamp,
                'X-Signature'    => $signature,
            ),
        );
        if ($body !== '') {
            $args['body'] = $body;
        }

        $response = wp_remote_request($this->hubUrl . $path, $args);
        if (is_wp_error($response)) {
            throw new Exception($response->get_error_message());
        }

        $code = wp_remote_retrieve_response_code($response);
        $raw  = wp_remote_retrieve_body($response);
        $data = json_decode($raw, true);
        if ($code >= 400 || !is_array($data) || empty($data['success'])) {
            $err = is_array($data) && !empty($data['error']) ? $data['error'] : ('HTTP ' . $code);
            throw new Exception('UPIPays: ' . $err);
        }
        return $data;
    }

    public function createOrder(array $payload)
    {
        return $this->request('POST', '/api/v1/orders/create', $payload);
    }

    public function verifyOrder($orderId)
    {
        $path = '/api/v1/orders/' . rawurlencode($orderId) . '/verify';
        return $this->request('GET', $path);
    }

    public function verifyWebhook($rawBody, $timestamp, $signature, $webhookPath)
    {
        $expected = $this->sign($timestamp, 'POST', $webhookPath, $rawBody);
        return hash_equals($expected, $signature);
    }
}
