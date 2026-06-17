package services

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
	"go.uber.org/zap"
)

type MerchantNotifier struct {
	merchants *repository.MerchantRepository
	log       *zap.Logger
}

func NewMerchantNotifier(merchants *repository.MerchantRepository, log *zap.Logger) *MerchantNotifier {
	return &MerchantNotifier{merchants: merchants, log: log}
}

func (n *MerchantNotifier) Notify(ctx context.Context, o *models.Order, event string) {
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

	bodyBytes, _ := json.Marshal(payload)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	whURL, _ := url.Parse(webhookURL)
	path := "/"
	if whURL != nil && whURL.Path != "" {
		path = whURL.Path
	}
	sig := security.Sign(merchant.APISecret, ts, http.MethodPost, path, string(bodyBytes))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Hub-Signature", sig)
	req.Header.Set("X-Hub-Timestamp", ts)
	req.Header.Set("X-Hub-Event", event)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		n.log.Error("notify merchant failed", zap.Error(err), zap.String("url", webhookURL))
		return
	}
	defer resp.Body.Close()
	n.log.Info("merchant webhook sent", zap.Int("status", resp.StatusCode), zap.String("event", event))
}

func (n *MerchantNotifier) NotifyAsync(o *models.Order, event string) {
	go n.Notify(context.Background(), o, event)
}
