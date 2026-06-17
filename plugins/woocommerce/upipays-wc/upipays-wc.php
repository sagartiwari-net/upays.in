<?php
/**
 * Plugin Name: UPIPays for WooCommerce
 * Plugin URI: https://upays.in/docs
 * Description: Accept UPI payments via UPIPays Dynamic QR — 0% transaction fee.
 * Version: 1.0.0
 * Author: UPIPays
 * Author URI: https://upays.in
 * Text Domain: upipays-wc
 * Requires at least: 5.8
 * Requires PHP: 7.4
 * WC requires at least: 6.0
 * WC tested up to: 9.0
 */

defined('ABSPATH') || exit;

define('UPIPAYS_WC_VERSION', '1.0.0');
define('UPIPAYS_WC_PLUGIN_FILE', __FILE__);
define('UPIPAYS_WC_PLUGIN_DIR', plugin_dir_path(__FILE__));

add_action('plugins_loaded', 'upipays_wc_init', 11);

function upipays_wc_init()
{
    if (!class_exists('WooCommerce')) {
        add_action('admin_notices', function () {
            echo '<div class="error"><p><strong>UPIPays</strong> requires WooCommerce.</p></div>';
        });
        return;
    }

    require_once UPIPAYS_WC_PLUGIN_DIR . 'includes/class-upipays-client.php';
    require_once UPIPAYS_WC_PLUGIN_DIR . 'includes/class-wc-gateway-upipays.php';

    add_filter('woocommerce_payment_gateways', function ($methods) {
        $methods[] = 'WC_Gateway_Upipays';
        return $methods;
    });
}

add_action('before_woocommerce_init', function () {
    if (class_exists('\Automattic\WooCommerce\Utilities\FeaturesUtil')) {
        \Automattic\WooCommerce\Utilities\FeaturesUtil::declare_compatibility('custom_order_tables', __FILE__, true);
    }
});
