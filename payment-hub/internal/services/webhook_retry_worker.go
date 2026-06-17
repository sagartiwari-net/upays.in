package services

import (
	"context"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"go.uber.org/zap"
)

type WebhookRetryWorker struct {
	webhooks *repository.WebhookLogRepo
	notifier *MerchantNotifier
	log      *zap.Logger
	interval time.Duration
}

func NewWebhookRetryWorker(
	webhooks *repository.WebhookLogRepo,
	notifier *MerchantNotifier,
	log *zap.Logger,
) *WebhookRetryWorker {
	return &WebhookRetryWorker{
		webhooks: webhooks,
		notifier: notifier,
		log:      log,
		interval: 30 * time.Second,
	}
}

func (w *WebhookRetryWorker) Start(ctx context.Context) {
	if w == nil || w.webhooks == nil || w.notifier == nil {
		return
	}
	w.log.Info("webhook retry worker started")
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("webhook retry worker stopped")
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *WebhookRetryWorker) runOnce(ctx context.Context) {
	items, err := w.webhooks.ListDueRetries(ctx, 20)
	if err != nil {
		w.log.Error("webhook retry: list due", zap.Error(err))
		return
	}
	for _, item := range items {
		w.notifier.RetryFromLog(ctx, item)
	}
}
