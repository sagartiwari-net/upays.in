<?php

/**
 * @title UPIPays
 * @description Pay via UPIPays Dynamic QR (UPI on upays.in)
 */

class Am_Paysystem_Upipays extends Am_Paysystem_Abstract
{
    const PLUGIN_STATUS = self::STATUS_PRODUCTION;
    const PLUGIN_REVISION = '1.0.0';

    protected $defaultTitle = 'UPIPays';
    protected $defaultDescription = 'Pay via UPIPays Dynamic QR (upays.in)';

    protected $_canResendPostback = true;

    function supportsCancelPage()
    {
        return false;
    }

    public function canUpgrade(Invoice $invoice, InvoiceItem $item, ProductUpgrade $upgrade)
    {
        return false;
    }

    public function getRecurringType()
    {
        return self::REPORTS_NOT_RECURRING;
    }

    public function _initSetupForm(Am_Form_Setup $form)
    {
        $form->addText('api_key')
            ->setLabel("Merchant API Key\n" .
                'From UPIPays dashboard → Manage Keys')
            ->addRule('required');

        $form->addSecretText('api_secret')
            ->setLabel("Merchant API Secret\n" .
                'From UPIPays dashboard → Manage Keys')
            ->addRule('required');

        $form->addText('hub_url', array('size' => 60))
            ->setLabel("Hub Base URL\n" .
                'Default: https://upays.in');
    }

    function isConfigured()
    {
        return strlen($this->getConfig('api_key'))
            && strlen($this->getConfig('api_secret'));
    }

    function getSupportedCurrencies()
    {
        return array('INR');
    }

    function getHubUrl()
    {
        $url = trim($this->getConfig('hub_url'));
        if ($url === '') {
            $url = 'https://upays.in';
        }
        return rtrim($url, '/');
    }

    function hubRequest($method, $path, $body = '')
    {
        $timestamp = (string) time();
        $message = $timestamp . '|' . strtoupper($method) . '|' . $path . '|' . $body;
        $signature = hash_hmac('sha256', $message, $this->getConfig('api_secret'));

        $req = new Am_HttpRequest($this->getHubUrl() . $path, $method);
        $req->setHeader('Content-Type', 'application/json');
        $req->setHeader('X-Merchant-Key', $this->getConfig('api_key'));
        $req->setHeader('X-Timestamp', $timestamp);
        $req->setHeader('X-Signature', $signature);
        if ($body !== '') {
            $req->setBody($body);
        }

        $res = $req->send();
        $decoded = json_decode($res->getBody(), true);
        if ($res->getStatus() >= 400 || !is_array($decoded) || empty($decoded['success'])) {
            $err = is_array($decoded) && !empty($decoded['error']) ? $decoded['error'] : ('HTTP ' . $res->getStatus());
            throw new Am_Exception_InputError('UPIPays API error: ' . $err);
        }
        return $decoded;
    }

    public function _process($invoice, $request, $result)
    {
        if (!$this->isConfigured()) {
            throw new Am_Exception_InputError('UPIPays plugin is not configured. Enter API Key and Secret in plugin settings.');
        }

        $user = $invoice->getUser();
        $productName = method_exists($invoice, 'getLineDescription') && $invoice->getLineDescription()
            ? $invoice->getLineDescription()
            : 'Membership';

        $customerName = $user->login;
        if (method_exists($user, 'getName') && trim($user->getName()) !== '') {
            $customerName = trim($user->getName());
        }

        $payload = array(
            'order_id' => $invoice->public_id,
            'amount' => (float) $invoice->first_total,
            'currency' => $invoice->currency ? $invoice->currency : 'INR',
            'customer' => array(
                'email' => $user->email,
                'name' => $customerName,
            ),
            'product' => array(
                'name' => $productName,
            ),
            'return_url' => $this->getReturnUrl(),
            'webhook_url' => $this->getPluginUrl('ipn'),
        );

        $body = json_encode($payload, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
        $response = $this->hubRequest(Am_HttpRequest::METHOD_POST, '/api/v1/orders/create', $body);

        $log = $this->getDi()->invoiceLogRecord;
        $log->setInvoice($invoice);
        $log->paysys_id = $invoice->paysys_id;
        $log->add($body);
        $log->add(json_encode($response));

        $paymentUrl = !empty($response['data']['payment_url']) ? $response['data']['payment_url'] : '';
        if ($paymentUrl === '') {
            throw new Am_Exception_InputError('UPIPays did not return payment_url');
        }

        $a = new Am_Paysystem_Action_Redirect($paymentUrl);
        $result->setAction($a);
    }

    public function createTransaction($request, $response, array $invokeArgs)
    {
        return new Am_Paysystem_Transaction_Upipays($this, $request, $response, $invokeArgs);
    }
}

class Am_Paysystem_Transaction_Upipays extends Am_Paysystem_Transaction_Incoming
{
    /** @var array */
    protected $vars;
    /** @var bool */
    protected $isWebhook = false;

    function __construct($plugin, $request, $response, $invokeArgs)
    {
        $raw = $request->getRawBody();
        $decoded = $raw ? json_decode($raw, true) : null;

        if (is_array($decoded) && !empty($decoded['event'])) {
            $this->isWebhook = true;
            $this->vars = $decoded;
        } else {
            $this->vars = array(
                'event' => 'payment.return',
                'order_id' => $request->getParam('order_id'),
                'status' => $request->getParam('status'),
                'hub_order_id' => $request->getParam('hub_order_id'),
            );
        }

        parent::__construct($plugin, $request, $response, $invokeArgs);
    }

    public function getUniqId()
    {
        if (!empty($this->vars['hub_order_id'])) {
            return $this->vars['hub_order_id'];
        }
        $invoiceId = $this->findInvoiceId();
        return 'upipays-' . ($invoiceId ? $invoiceId : uniqid());
    }

    function findInvoiceId()
    {
        return !empty($this->vars['order_id']) ? $this->vars['order_id'] : null;
    }

    public function validateSource()
    {
        if (!$this->isWebhook) {
            return true;
        }

        $raw = $this->request->getRawBody();
        $timestamp = (string) $this->request->getHeader('X-Hub-Timestamp');
        $signature = (string) $this->request->getHeader('X-Hub-Signature');
        if ($timestamp === '' || $signature === '') {
            return false;
        }

        $webhookPath = parse_url($this->plugin->getPluginUrl('ipn'), PHP_URL_PATH);
        if (!$webhookPath) {
            $webhookPath = '/payment/upipays/ipn';
        }

        $message = $timestamp . '|POST|' . $webhookPath . '|' . $raw;
        $expected = hash_hmac('sha256', $message, $this->plugin->getConfig('api_secret'));
        return hash_equals($expected, $signature);
    }

    public function validateStatus()
    {
        if ($this->isWebhook) {
            return !empty($this->vars['event'])
                && $this->vars['event'] === 'payment.success'
                && !empty($this->vars['status'])
                && $this->vars['status'] === 'success';
        }

        $orderId = $this->findInvoiceId();
        if (!$orderId) {
            return false;
        }

        try {
            $verified = $this->plugin->hubRequest(
                Am_HttpRequest::METHOD_GET,
                '/api/v1/orders/' . rawurlencode($orderId) . '/verify'
            );
            return !empty($verified['data']['status']) && $verified['data']['status'] === 'success';
        } catch (Exception $e) {
            return false;
        }
    }

    public function validateTerms()
    {
        return true;
    }

    function processValidated()
    {
        $this->invoice->addPayment($this);
    }
}
