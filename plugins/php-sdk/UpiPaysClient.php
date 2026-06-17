<?php
/**
 * UPIPays PHP client — HMAC-signed API requests.
 * Works standalone (curl) or inside WordPress (wp_remote_request).
 */

class UpiPays_Client
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
            $body = json_encode($payload, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
        }
        $timestamp = (string) time();
        $signature = $this->sign($timestamp, $method, $path, $body);

        $headers = array(
            'Content-Type: application/json',
            'X-Merchant-Key: ' . $this->apiKey,
            'X-Timestamp: ' . $timestamp,
            'X-Signature: ' . $signature,
        );

        if (function_exists('wp_remote_request')) {
            return $this->requestWordPress($method, $path, $body, $headers);
        }
        return $this->requestCurl($method, $path, $body, $headers);
    }

    private function requestWordPress($method, $path, $body, $headers)
    {
        $h = array();
        foreach ($headers as $line) {
            $parts = explode(': ', $line, 2);
            if (count($parts) === 2) {
                $h[$parts[0]] = $parts[1];
            }
        }
        $args = array('method' => strtoupper($method), 'timeout' => 30, 'headers' => $h);
        if ($body !== '') {
            $args['body'] = $body;
        }
        $response = wp_remote_request($this->hubUrl . $path, $args);
        if (is_wp_error($response)) {
            throw new Exception($response->get_error_message());
        }
        $code = wp_remote_retrieve_response_code($response);
        $raw  = wp_remote_retrieve_body($response);
        return $this->parseResponse($code, $raw);
    }

    private function requestCurl($method, $path, $body, $headers)
    {
        $ch = curl_init($this->hubUrl . $path);
        curl_setopt_array($ch, array(
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT        => 30,
            CURLOPT_CUSTOMREQUEST  => strtoupper($method),
            CURLOPT_HTTPHEADER     => $headers,
        ));
        if ($body !== '') {
            curl_setopt($ch, CURLOPT_POSTFIELDS, $body);
        }
        $raw = curl_exec($ch);
        $code = (int) curl_getinfo($ch, CURLINFO_HTTP_CODE);
        if ($raw === false) {
            $err = curl_error($ch);
            curl_close($ch);
            throw new Exception($err);
        }
        curl_close($ch);
        return $this->parseResponse($code, $raw);
    }

    private function parseResponse($code, $raw)
    {
        $data = json_decode($raw, true);
        if ($code >= 400 || !is_array($data) || empty($data['success'])) {
            $err = is_array($data) && !empty($data['error']) ? $data['error'] : ('HTTP ' . $code);
            throw new Exception('UPIPays API: ' . $err);
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
