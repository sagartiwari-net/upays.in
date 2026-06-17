package services

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/skip2/go-qrcode"
	"go.uber.org/zap"
)

var utrPattern = regexp.MustCompile(`^\d{10,20}$`)

type UPIService struct {
	orders    *repository.OrderRepository
	merchants *repository.MerchantRepository
	profiles  *ProfileService
	notifier  *MerchantNotifier
	appURL    string
	poller    EmailPoller
	log       *zap.Logger
}

func NewUPIService(
	orders *repository.OrderRepository,
	merchants *repository.MerchantRepository,
	profiles *ProfileService,
	notifier *MerchantNotifier,
	appURL string,
	poller EmailPoller,
	log *zap.Logger,
) *UPIService {
	return &UPIService{
		orders:    orders,
		merchants: merchants,
		profiles:  profiles,
		notifier:  notifier,
		appURL:    strings.TrimRight(appURL, "/"),
		poller:    poller,
		log:       log,
	}
}

type UPICheckoutView struct {
	Token        string
	HubOrderID   string
	ProductName  string
	Amount       string
	PayAmount    string
	Merchant     string
	UPIID        string
	QRImageURL   string
	Status       string
	Expired      bool
	Payable      bool
	ReturnURL       string
	ExpiresAtUnixMs int64
	ExpiryMinutes   int
}

func (s *UPIService) CheckoutURL(token string) string {
	return fmt.Sprintf("%s/pay/%s", s.appURL, token)
}

func (s *UPIService) applyTimeout(ctx context.Context, o *models.Order) *models.Order {
	if o.Status == models.OrderStatusPending && time.Now().UTC().After(o.ExpiresAt) {
		_ = s.orders.TransitionStatus(ctx, o.ID, models.OrderStatusPending, models.OrderStatusFailed)
		o.Status = models.OrderStatusFailed
	}
	return o
}

func (s *UPIService) GetCheckout(ctx context.Context, token string) (*UPICheckoutView, *models.Order, error) {
	o, err := s.orders.GetByPaymentToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	merchant, err := s.merchants.GetByID(ctx, o.MerchantID)
	if err != nil {
		return nil, nil, err
	}

	o = s.applyTimeout(ctx, o)

	profile, err := s.profiles.ResolveForOrder(ctx, o)
	if err != nil {
		return nil, nil, err
	}

	expiresAt := o.ExpiresAt.UTC()
	if expiresAt.IsZero() || expiresAt.Year() < 2000 {
		expiresAt = o.CreatedAt.UTC().Add(5 * time.Minute)
	}
	expiryMinutes := int(time.Until(expiresAt).Minutes())
	if expiryMinutes < 1 {
		expiryMinutes = 5
	}

	view := &UPICheckoutView{
		Token:           token,
		HubOrderID:      o.HubOrderID,
		ProductName:     fallbackStr(o.ProductName, "Payment"),
		Amount:          fmt.Sprintf("%.2f", o.Amount),
		PayAmount:       fmt.Sprintf("%.2f", o.PayAmount),
		Merchant:        merchant.Domain,
		UPIID:           profile.UPIID,
		QRImageURL:      fmt.Sprintf("%s/pay/%s/qr", s.appURL, token),
		Status:          o.Status,
		Expired:         o.Status == models.OrderStatusFailed,
		Payable:         o.Status == models.OrderStatusPending,
		ReturnURL:       s.buildReturnURL(o),
		ExpiresAtUnixMs: expiresAt.UnixMilli(),
		ExpiryMinutes:   expiryMinutes,
	}
	return view, o, nil
}

func (s *UPIService) GetQRImage(ctx context.Context, token string) ([]byte, error) {
	o, err := s.orders.GetByPaymentToken(ctx, token)
	if err != nil {
		return nil, err
	}
	profile, err := s.profiles.ResolveForOrder(ctx, o)
	if err != nil {
		return nil, err
	}
	return qrcode.Encode(s.upiURI(o, profile), qrcode.Medium, 256)
}

func (s *UPIService) upiURI(o *models.Order, profile *models.PaymentProfile) string {
	q := url.Values{}
	q.Set("pa", profile.UPIID)
	q.Set("pn", profile.PayeeName)
	q.Set("am", fmt.Sprintf("%.2f", o.PayAmount))
	q.Set("cu", "INR")
	q.Set("tn", o.HubOrderID)
	return "upi://pay?" + q.Encode()
}

type PaymentStatusResponse struct {
	Status      string  `json:"status"`
	RedirectURL *string `json:"redirect_url"`
}

func (s *UPIService) GetStatus(ctx context.Context, token string) (*PaymentStatusResponse, error) {
	o, err := s.orders.GetByPaymentToken(ctx, token)
	if err != nil {
		return nil, err
	}

	o = s.applyTimeout(ctx, o)

	resp := &PaymentStatusResponse{Status: o.Status}
	if o.Status == models.OrderStatusSuccess || o.Status == models.OrderStatusFailed {
		u := s.buildReturnURL(o)
		resp.RedirectURL = &u
	}
	return resp, nil
}

func (s *UPIService) SubmitUTR(ctx context.Context, token, utr string) error {
	utr = strings.TrimSpace(utr)
	if !utrPattern.MatchString(utr) {
		return ErrInvalidInput
	}

	o, err := s.orders.GetByPaymentToken(ctx, token)
	if err != nil {
		return err
	}
	o = s.applyTimeout(ctx, o)
	if o.Status != models.OrderStatusPending {
		if o.Status == models.OrderStatusSuccess {
			return nil
		}
		return ErrOrderNotPayable
	}
	if time.Now().UTC().After(o.ExpiresAt) {
		return ErrOrderExpired
	}

	if err := s.orders.SetCustomerUTR(ctx, o.ID, utr); err != nil {
		return err
	}

	if s.poller != nil {
		s.poller.TriggerPoll()
	}
	return nil
}

func (s *UPIService) buildReturnURL(o *models.Order) string {
	u, err := url.Parse(o.ReturnURL)
	if err != nil {
		return o.ReturnURL
	}
	q := u.Query()
	q.Set("order_id", o.MerchantOrderID)
	q.Set("status", o.Status)
	q.Set("hub_order_id", o.HubOrderID)
	u.RawQuery = q.Encode()
	return u.String()
}

func fallbackStr(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

var upiCheckoutTemplate = template.Must(template.New("upi_checkout").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Pay via UPI</title>
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    :root {
      --primary-dark: #1a1a2e;
      --primary-purple: #6c63ff;
      --accent-green: #4ade80;
      --bg-light: #f8f9fa;
      --bg-white: #ffffff;
      --text-secondary: #6b7280;
      --border-color: #e5e7eb;
      --radius-md: 12px;
      --radius-xl: 20px;
      --shadow-lg: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
    }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background: linear-gradient(135deg, #f8f9fa 0%, #eef2ff 100%);
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px 16px;
      color: var(--primary-dark);
    }
    .wrap { width: 100%; max-width: 440px; }
    .brand {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 10px;
      margin-bottom: 20px;
    }
    .brand-icon {
      width: 36px; height: 36px;
      background: linear-gradient(135deg, var(--primary-purple), #9333ea);
      border-radius: 8px;
      display: flex; align-items: center; justify-content: center;
      color: white; font-size: 16px;
    }
    .brand-text { font-size: 20px; font-weight: 700; }
    .card {
      background: var(--bg-white);
      border-radius: var(--radius-xl);
      padding: 28px 24px;
      box-shadow: var(--shadow-lg);
      border: 1px solid var(--border-color);
      text-align: center;
    }
    h1 { font-size: 20px; font-weight: 700; margin-bottom: 6px; }
    .muted { color: var(--text-secondary); font-size: 13px; margin-bottom: 20px; }
    .timer {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      background: rgba(239, 68, 68, 0.08);
      color: #dc2626;
      padding: 8px 16px;
      border-radius: 20px;
      font-size: 13px;
      font-weight: 600;
      margin-bottom: 16px;
    }
    .timer strong { font-size: 16px; font-variant-numeric: tabular-nums; }
    .amount {
      font-size: 36px;
      font-weight: 700;
      background: linear-gradient(135deg, var(--primary-purple), #9333ea);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
      background-clip: text;
      margin: 8px 0 4px;
    }
    .note { font-size: 12px; color: var(--text-secondary); margin-bottom: 20px; }
    .qr-wrap {
      background: var(--bg-light);
      border-radius: var(--radius-md);
      padding: 16px;
      margin-bottom: 16px;
      display: inline-block;
    }
    .qr { width: 200px; height: 200px; border-radius: 8px; display: block; }
    .upi-id {
      font-size: 14px;
      background: var(--bg-light);
      padding: 12px 16px;
      border-radius: var(--radius-md);
      margin-bottom: 20px;
      word-break: break-all;
      border: 1px dashed var(--border-color);
    }
    .upi-id strong { color: var(--primary-purple); }
    .steps {
      text-align: left;
      font-size: 14px;
      color: #374151;
      margin: 16px 0;
      line-height: 1.7;
      background: var(--bg-light);
      padding: 16px;
      border-radius: var(--radius-md);
    }
    .steps ol { padding-left: 20px; margin-top: 8px; }
    .status-msg { margin-top: 16px; font-size: 14px; color: var(--text-secondary); }
    .utr-box {
      margin-top: 20px;
      padding-top: 20px;
      border-top: 1px solid var(--border-color);
      text-align: left;
    }
    .utr-box label { font-size: 13px; color: var(--text-secondary); display: block; margin-bottom: 8px; font-weight: 600; }
    .utr-box input {
      width: 100%;
      padding: 12px 16px;
      border: 1px solid var(--border-color);
      border-radius: var(--radius-md);
      font-size: 15px;
      outline: none;
    }
    .utr-box input:focus { border-color: var(--primary-purple); box-shadow: 0 0 0 3px rgba(108,99,255,0.1); }
    .utr-box button {
      margin-top: 10px;
      width: 100%;
      padding: 12px;
      border: 0;
      border-radius: var(--radius-md);
      background: linear-gradient(135deg, var(--primary-purple), #9333ea);
      color: #fff;
      font-size: 14px;
      font-weight: 600;
      cursor: pointer;
    }
    .success { color: #059669; font-weight: 600; }
    .fail { color: #dc2626; }
    .secure { text-align: center; margin-top: 16px; font-size: 12px; color: var(--text-secondary); }
    .secure i { color: var(--accent-green); margin-right: 4px; }
  </style>
</head>
<body>
  <div class="wrap">
    <div class="brand">
      <div class="brand-icon"><i class="fas fa-rocket"></i></div>
      <span class="brand-text">UPIPays</span>
    </div>
    <div class="card">
      <h1>{{.ProductName}}</h1>
      <div class="muted">via {{.Merchant}} · secured by upays.in</div>
      {{if .Payable}}
      <div class="timer" id="countdown"><i class="fas fa-clock"></i> Pay within <strong id="timer">05:00</strong></div>
      <div class="amount">₹{{.PayAmount}}</div>
      <div class="note">Pay exact amount shown (includes unique verification paise)</div>
      <div class="qr-wrap"><img class="qr" src="{{.QRImageURL}}" alt="UPI QR Code"></div>
      <div class="upi-id"><strong>UPI ID:</strong> {{.UPIID}}</div>
      <div class="steps">
        <strong>How to pay:</strong>
        <ol>
          <li>Open any UPI app (PhonePe, GPay, Paytm)</li>
          <li>Scan QR code above</li>
          <li>Pay exact ₹{{.PayAmount}} — do not change amount</li>
          <li>Complete within <strong id="timer-steps">{{.ExpiryMinutes}} minutes</strong></li>
        </ol>
      </div>
      <div class="status-msg" id="status-msg"><i class="fas fa-spinner fa-spin"></i> Waiting for payment…</div>
      <div class="utr-box">
        <label for="utr">Already paid? Enter UTR for faster verification (optional)</label>
        <input type="text" id="utr" placeholder="12-digit UTR" maxlength="20" inputmode="numeric">
        <button type="button" id="utr-btn">Submit UTR</button>
        <div id="utr-msg" class="muted" style="margin-top:8px;font-size:12px;"></div>
      </div>
      {{else if eq .Status "success"}}
      <div class="amount success"><i class="fas fa-check-circle"></i> Payment Successful</div>
      <p class="muted">Redirecting you back…</p>
      {{else if or .Expired (eq .Status "failed")}}
      <div class="amount fail"><i class="fas fa-times-circle"></i> Payment Failed</div>
      <p class="muted">Time limit exceeded. No payment was received. Redirecting you back…</p>
      {{else}}
      <p class="muted">This payment is no longer available.</p>
      {{end}}
    </div>
    <div class="secure"><i class="fas fa-shield-alt"></i> Secured by UPIPays</div>
  </div>
  {{if .Payable}}
  <script>
    const pathBase = window.location.pathname.replace(/\/$/, '');
    const statusURL = pathBase + '/status';
    const utrURL = pathBase + '/utr';
    const returnURL = {{printf "%q" .ReturnURL}};
    const expiresAt = {{.ExpiresAtUnixMs}} || (Date.now() + {{.ExpiryMinutes}} * 60 * 1000);

    function redirectBack(url) {
      const dest = url || returnURL;
      if (dest) window.location.replace(dest);
    }

    function updateCountdown() {
      const left = expiresAt - Date.now();
      if (!Number.isFinite(left) || left <= 0) {
        document.getElementById('timer').textContent = '00:00';
        handleTimeout();
        return;
      }
      const m = Math.floor(left / 60000);
      const s = Math.floor((left % 60000) / 1000);
      const txt = String(m).padStart(2, '0') + ':' + String(s).padStart(2, '0');
      document.getElementById('timer').textContent = txt;
      const steps = document.getElementById('timer-steps');
      if (steps) steps.textContent = txt;
      setTimeout(updateCountdown, 1000);
    }
    updateCountdown();

    async function handleTimeout() {
      document.getElementById('status-msg').textContent = 'Time limit exceeded. Redirecting…';
      try {
        const res = await fetch(statusURL, {cache: 'no-store'});
        if (res.ok) {
          const data = await res.json();
          redirectBack(data.redirect_url);
          return;
        }
      } catch (e) {}
      redirectBack(returnURL);
    }

    async function pollStatus() {
      try {
        const res = await fetch(statusURL, {cache: 'no-store'});
        if (!res.ok) throw new Error('status ' + res.status);
        const data = await res.json();
        if (data.status === 'success') {
          document.getElementById('status-msg').textContent = 'Payment verified! Redirecting…';
          document.getElementById('status-msg').className = 'status-msg success';
          redirectBack(data.redirect_url);
          return;
        }
        if (data.status === 'failed') {
          document.getElementById('status-msg').textContent = 'Payment failed — time limit exceeded.';
          redirectBack(data.redirect_url);
          return;
        }
      } catch (e) {
        console.warn('status poll failed', e);
      }
      setTimeout(pollStatus, 3000);
    }
    pollStatus();

    document.getElementById('utr-btn').addEventListener('click', async () => {
      const utr = document.getElementById('utr').value.trim();
      const msg = document.getElementById('utr-msg');
      if (!utr) { msg.textContent = 'Enter UTR number'; return; }
      try {
        const res = await fetch(utrURL, {
          method: 'POST',
          headers: {'Content-Type': 'application/json'},
          body: JSON.stringify({utr})
        });
        if (res.ok) {
          msg.textContent = 'UTR submitted — checking payment…';
          pollStatus();
        } else {
          const err = await res.json().catch(() => ({}));
          msg.textContent = err.error || 'Could not submit UTR. Try again.';
        }
      } catch (e) {
        msg.textContent = 'Network error. Try again.';
      }
    });
  </script>
  {{else if or .Expired (eq .Status "failed")}}
  <script>
    setTimeout(function() {
      window.location.replace({{printf "%q" .ReturnURL}});
    }, 2000);
  </script>
  {{else if eq .Status "success"}}
  <script>
    setTimeout(function() {
      window.location.replace({{printf "%q" .ReturnURL}});
    }, 1500);
  </script>
  {{end}}
</body>
</html>`))

func RenderUPICheckoutHTML(view *UPICheckoutView) (string, error) {
	var buf bytes.Buffer
	if err := upiCheckoutTemplate.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}
