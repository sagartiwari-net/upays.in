# UPIPays for WooCommerce

Accept UPI payments on your WooCommerce store via [UPIPays](https://upays.in) Dynamic QR.

## Requirements

- WordPress 5.8+
- WooCommerce 6.0+
- PHP 7.4+
- UPIPays merchant account ([register free](https://upays.in/dashboard/register))

## Install

1. Download `upipays-wc.zip` from UPIPays dashboard → **Integrations** or admin downloads
2. WordPress Admin → **Plugins → Add New → Upload Plugin**
3. Activate **UPIPays for WooCommerce**

## Configure

1. **WooCommerce → Settings → Payments → UPIPays UPI → Manage**
2. Enable the gateway
3. Enter:
   - **Hub URL:** `https://upays.in`
   - **API Key** and **API Secret** from [UPIPays dashboard](https://upays.in/dashboard/settings)
4. Save — plugin runs a ₹1 test API call to verify credentials

## Webhook

Webhook URL (auto-sent per order):

```
https://yoursite.com/?wc-api=wc_gateway_upipays
```

Successful UPI payments mark the WooCommerce order as **Processing/Completed** automatically.

## Test checkout

1. Create a ₹1 test product
2. Checkout with **Pay via UPI**
3. Complete payment on UPIPays QR page
4. Order status updates via webhook

## Support

- Docs: https://upays.in/docs
- Email: support@upays.in
