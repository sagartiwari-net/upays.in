package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
	"go.uber.org/zap"
)

const maxWebhookResponseBody = 4096

type MerchantNotifier struct {
	merchants *repository.MerchantRepository
	orders    *repository.OrderRepository
	webhooks  *repository.WebhookLogRepo
	log       *zap.Logger
}

func NewMerchantNotifier(
	merchants *repository.MerchantRepository,
	orders *repository.OrderRepository,
	webhooks *repository.WebhookLogRepo,
	log *zap.Logger,
) *MerchantNotifier {
	return &MerchantNotifier{merchants: merchants, orders: orders, webhooks: webhooks, log: log}
}

func (n *MerchantNotifier) Notify(ctx context.Context, o *models.Order, event string) {
	if n == nil || o == nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	merchant, err := n.merchants.GetByID(ctx, o.MerchantID)
	if err != nil {
		n.log.Error("notify merchant: load merchant", zap.Error(err))
		return
	}

	webhookURL := o.WebhookURL
	if webhookURL == "" {
		webhookURL = merchant.WebhookURL
	}
	if webhookURL == "" {
		return
	}

	bodyBytes, _ := json.Marshal(buildWebhookPayload(o, event))

	logID := ""
	if n.webhooks != nil {
		logID, err = n.webhooks.CreateOutbound(ctx, o.ID, o.MerchantID, string(bodyBytes))
		if err != nil {
			n.log.Error("notify merchant: create webhook log", zap.Error(err))
		}
	}

	n.deliver(ctx, logID, webhookURL, merchant.APISecret, bodyBytes, event, 0)
}

func (n *MerchantNotifier) RetryFromLog(ctx context.Context, rec repository.WebhookLogRecord) {
	if n == nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	merchant, err := n.merchants.GetByID(ctx, rec.MerchantID)
	if err != nil {
		n.log.Error("webhook retry: load merchant", zap.Error(err))
		return
	}

	webhookURL := merchant.WebhookURL
	if order, err := n.orders.GetByID(ctx, rec.OrderID); err == nil && order.WebhookURL != "" {
		webhookURL = order.WebhookURL
	}
	if webhookURL == "" {
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(rec.Payload), &payload); err != nil {
		n.log.Error("webhook retry: bad payload", zap.Error(err), zap.String("id", rec.ID))
		return
	}
	event, _ := payload["event"].(string)

	n.deliver(ctx, rec.ID, webhookURL, merchant.APISecret, []byte(rec.Payload), event, rec.RetryCount)
}

func (n *MerchantNotifier) deliver(
	ctx context.Context,
	logID, webhookURL, apiSecret string,
	bodyBytes []byte,
	event string,
	currentRetryCount int,
) {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	whURL, _ := url.Parse(webhookURL)
	path := "/"
	if whURL != nil && whURL.Path != "" {
		path = whURL.Path
	}
	sig := security.Sign(apiSecret, ts, http.MethodPost, path, string(bodyBytes))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		n.recordFailure(ctx, logID, nil, err.Error(), currentRetryCount)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature", sig)
	req.Header.Set("X-Hub-Timestamp", ts)
	req.Header.Set("X-Hub-Event", event)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		n.log.Error("notify merchant failed", zap.Error(err), zap.String("url", webhookURL))
		n.recordFailure(ctx, logID, nil, err.Error(), currentRetryCount)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxWebhookResponseBody))
	respStr := string(respBody)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if logID != "" && n.webhooks != nil {
			if err := n.webhooks.MarkDelivered(ctx, logID, resp.StatusCode, respStr); err != nil {
				n.log.Error("webhook log: mark delivered", zap.Error(err))
			}
		}
		n.log.Info("merchant webhook sent", zap.Int("status", resp.StatusCode), zap.String("event", event))
		return
	}

	n.log.Warn("merchant webhook non-2xx", zap.Int("status", resp.StatusCode), zap.String("event", event))
	code := resp.StatusCode
	n.recordFailure(ctx, logID, &code, respStr, currentRetryCount)
}

func (n *MerchantNotifier) recordFailure(ctx context.Context, logID string, code *int, body string, currentRetryCount int) {
	if logID == "" || n.webhooks == nil {
		return
	}
	nextCount := currentRetryCount + 1
	nextAt := time.Now().UTC().Add(repository.WebhookRetryDelay(currentRetryCount))
	if err := n.webhooks.ScheduleRetry(ctx, logID, code, body, nextCount, nextAt); err != nil {
		n.log.Error("webhook log: schedule retry", zap.Error(err))
	}
}

func buildWebhookPayload(o *models.Order, event string) map[string]interface{} {
	payload := map[string]interface{}{
		"event":        event,
		"hub_order_id": o.HubOrderID,
		"order_id":     o.MerchantOrderID,
		"amount":       o.Amount,
		"currency":     o.Currency,
		"status":       o.Status,
	}
	if o.PhonePeTxnID != "" {
		payload["bank_txn_id"] = o.PhonePeTxnID
	}
	if o.PaidAt != nil {
		payload["paid_at"] = o.PaidAt.UTC().Format(time.RFC3339)
	}
	return payload
}

func (n *MerchantNotifier) NotifyAsync(o *models.Order, event string) {
	go n.Notify(context.Background(), o, event)
}
