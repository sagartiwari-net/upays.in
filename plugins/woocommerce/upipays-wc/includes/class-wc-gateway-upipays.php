<?php
defined('ABSPATH') || exit;

class WC_Gateway_Upipays extends WC_Payment_Gateway
{
    public function __construct()
    {
        $this->id                 = 'upipays';
        $this->icon               = '';
        $this->has_fields         = false;
        $this->method_title       = __('UPIPays UPI', 'upipays-wc');
        $this->method_description = __('Pay via UPI Dynamic QR (upays.in). 0% transaction fee.', 'upipays-wc');
        $this->supports           = array('products');

        $this->init_form_fields();
        $this->init_settings();

        $this->title       = $this->get_option('title', 'Pay via UPI');
        $this->description = $this->get_option('description', 'Scan QR and pay with any UPI app.');
        $this->enabled     = $this->get_option('enabled', 'no');

        add_action('woocommerce_update_options_payment_gateways_' . $this->id, array($this, 'process_admin_options'));
        add_action('woocommerce_api_wc_gateway_upipays', array($this, 'webhook_handler'));
        add_action('admin_notices', array($this, 'admin_notices'));
    }

    public function init_form_fields()
    {
        $webhook = home_url('/?wc-api=wc_gateway_upipays');

        $this->form_fields = array(
            'enabled' => array(
                'title'   => __('Enable/Disable', 'upipays-wc'),
                'type'    => 'checkbox',
                'label'   => __('Enable UPIPays UPI payments', 'upipays-wc'),
                'default' => 'no',
            ),
            'title' => array(
                'title'       => __('Title', 'upipays-wc'),
                'type'        => 'text',
                'description' => __('Checkout payment method title.', 'upipays-wc'),
                'default'     => 'Pay via UPI',
            ),
            'description' => array(
                'title'       => __('Description', 'upipays-wc'),
                'type'        => 'textarea',
                'description' => __('Shown on checkout.', 'upipays-wc'),
                'default'     => 'Pay securely with PhonePe, Google Pay, Paytm, or any UPI app.',
            ),
            'hub_url' => array(
                'title'       => __('Hub URL', 'upipays-wc'),
                'type'        => 'text',
                'description' => __('Default: https://upays.in', 'upipays-wc'),
                'default'     => 'https://upays.in',
            ),
            'api_key' => array(
                'title'       => __('API Key', 'upipays-wc'),
                'type'        => 'text',
                'description' => __('From UPIPays dashboard → Settings', 'upipays-wc'),
            ),
            'api_secret' => array(
                'title'       => __('API Secret', 'upipays-wc'),
                'type'        => 'password',
                'description' => __('Keep secret. From UPIPays dashboard → Settings', 'upipays-wc'),
            ),
            'webhook_info' => array(
                'title'       => __('Webhook URL', 'upipays-wc'),
                'type'        => 'title',
                'description' => sprintf(
                    __('Add this URL in UPIPays dashboard webhook settings (optional — sent per order automatically): %s', 'upipays-wc'),
                    '<code>' . esc_html($webhook) . '</code>'
                ),
            ),
        );
    }

    public function admin_notices()
    {
        if ($this->enabled !== 'yes' || $this->is_configured()) {
            return;
        }
        echo '<div class="notice notice-warning"><p><strong>UPIPays:</strong> Enter API Key and Secret in WooCommerce → Settings → Payments → UPIPays.</p></div>';
    }

    public function is_configured()
    {
        return $this->get_option('api_key') && $this->get_option('api_secret');
    }

    public function get_client()
    {
        $hub = $this->get_option('hub_url');
        if (!$hub) {
            $hub = 'https://upays.in';
        }
        return new UpiPays_WC_Client($hub, $this->get_option('api_key'), $this->get_option('api_secret'));
    }

    public function process_payment($order_id)
    {
        $order = wc_get_order($order_id);
        if (!$order) {
            wc_add_notice(__('Order not found.', 'upipays-wc'), 'error');
            return array('result' => 'fail');
        }

        if (!$this->is_configured()) {
            wc_add_notice(__('UPIPays is not configured.', 'upipays-wc'), 'error');
            return array('result' => 'fail');
        }

        try {
            $client = $this->get_client();
            $webhookPath = wp_parse_url(home_url('/?wc-api=wc_gateway_upipays'), PHP_URL_PATH);
            if (!$webhookPath) {
                $webhookPath = '/';
            }

            $payload = array(
                'order_id'    => (string) $order->get_id(),
                'amount'      => (float) $order->get_total(),
                'currency'    => $order->get_currency() ?: 'INR',
                'customer'    => array(
                    'email' => $order->get_billing_email(),
                    'name'  => trim($order->get_billing_first_name() . ' ' . $order->get_billing_last_name()),
                    'phone' => $order->get_billing_phone(),
                ),
                'product'     => array(
                    'name' => sprintf(__('Order #%s', 'upipays-wc'), $order->get_order_number()),
                ),
                'return_url'  => $this->get_return_url($order),
                'webhook_url' => home_url('/?wc-api=wc_gateway_upipays'),
            );

            $response = $client->createOrder($payload);
            $paymentUrl = isset($response['data']['payment_url']) ? $response['data']['payment_url'] : '';

            if ($paymentUrl === '') {
                throw new Exception('No payment_url returned');
            }

            $order->update_meta_data('_upipays_hub_order_id', $response['data']['hub_order_id'] ?? '');
            $order->update_status('pending', __('Awaiting UPI payment via UPIPays.', 'upipays-wc'));
            $order->save();

            WC()->cart->empty_cart();

            return array(
                'result'   => 'success',
                'redirect' => $paymentUrl,
            );
        } catch (Exception $e) {
            wc_add_notice($e->getMessage(), 'error');
            return array('result' => 'fail');
        }
    }

    public function webhook_handler()
    {
        $raw = file_get_contents('php://input');
        $timestamp = isset($_SERVER['HTTP_X_HUB_TIMESTAMP']) ? sanitize_text_field(wp_unslash($_SERVER['HTTP_X_HUB_TIMESTAMP'])) : '';
        $signature = isset($_SERVER['HTTP_X_HUB_SIGNATURE']) ? sanitize_text_field(wp_unslash($_SERVER['HTTP_X_HUB_SIGNATURE'])) : '';

        if (!$this->is_configured()) {
            status_header(500);
            exit;
        }

        $webhookPath = wp_parse_url(home_url('/?wc-api=wc_gateway_upipays'), PHP_URL_PATH);
        if (!$webhookPath) {
            $webhookPath = '/';
        }

        $client = $this->get_client();
        if (!$client->verifyWebhook($raw, $timestamp, $signature, $webhookPath)) {
            status_header(401);
            exit;
        }

        $data = json_decode($raw, true);
        if (!is_array($data) || empty($data['order_id'])) {
            status_header(400);
            exit;
        }

        $order = wc_get_order($data['order_id']);
        if (!$order) {
            status_header(404);
            exit;
        }

        if (!empty($data['event']) && $data['event'] === 'payment.success' && !empty($data['status']) && $data['status'] === 'success') {
            if (!$order->is_paid()) {
                $order->payment_complete($data['hub_order_id'] ?? '');
                $order->add_order_note(__('UPI payment confirmed via UPIPays webhook.', 'upipays-wc'));
            }
        }

        status_header(200);
        echo 'OK';
        exit;
    }

    public function process_admin_options()
    {
        parent::process_admin_options();
        if ($this->enabled === 'yes' && $this->is_configured()) {
            try {
                $client = $this->get_client();
                $testId = 'WC-TEST-' . time();
                $resp = $client->createOrder(array(
                    'order_id'   => $testId,
                    'amount'     => 1.0,
                    'currency'   => 'INR',
                    'customer'   => array('email' => get_bloginfo('admin_email'), 'name' => 'Test'),
                    'product'    => array('name' => 'UPIPays connection test'),
                    'return_url' => home_url('/'),
                    'webhook_url'=> home_url('/?wc-api=wc_gateway_upipays'),
                ));
                if (!empty($resp['data']['payment_url'])) {
                    WC_Admin_Settings::add_message(__('UPIPays API connection OK. Test payment URL created (₹1) — check UPIPays dashboard for pending order.', 'upipays-wc'));
                }
            } catch (Exception $e) {
                WC_Admin_Settings::add_error(__('UPIPays API test failed: ', 'upipays-wc') . $e->getMessage());
            }
        }
    }
}
