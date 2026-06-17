package emailverify

import (
	"context"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/services"
	"go.uber.org/zap"
)

type ProfilePoller interface {
	TriggerPoll()
}

type MultiProfileWorker struct {
	profiles *repository.PaymentProfileRepository
	matcher  *Matcher
	orders   *repository.OrderRepository
	interval time.Duration
	log      *zap.Logger
	trigger  chan struct{}
}

func NewMultiProfileWorker(
	profiles *repository.PaymentProfileRepository,
	matcher *Matcher,
	orders *repository.OrderRepository,
	interval time.Duration,
	log *zap.Logger,
) *MultiProfileWorker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &MultiProfileWorker{
		profiles: profiles,
		matcher:  matcher,
		orders:   orders,
		interval: interval,
		log:      log,
		trigger:  make(chan struct{}, 1),
	}
}

func (w *MultiProfileWorker) TriggerPoll() {
	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

func (w *MultiProfileWorker) Start(ctx context.Context) {
	w.log.Info("multi-profile email worker started", zap.Duration("interval", w.interval))
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	w.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			w.log.Info("email verification worker stopped")
			return
		case <-ticker.C:
			w.poll(ctx)
		case <-w.trigger:
			w.poll(ctx)
		}
	}
}

func (w *MultiProfileWorker) poll(ctx context.Context) {
	if _, err := w.orders.ExpirePendingOrders(ctx); err != nil {
		w.log.Warn("expire pending orders", zap.Error(err))
	}

	profiles, err := w.profiles.List(ctx, true)
	if err != nil {
		w.log.Error("list profiles", zap.Error(err))
		return
	}

	for _, profile := range profiles {
		if profile.IMAPUser == "" || profile.IMAPPassword == "" {
			continue
		}
		w.pollProfile(ctx, profile)
	}
}

func (w *MultiProfileWorker) pollProfile(ctx context.Context, profile *models.PaymentProfile) {
	imap := NewIMAPClient(profile.IMAPHost, profile.IMAPPort, profile.IMAPUser, profile.IMAPPassword, profile.SenderFilter)
	since := time.Now().UTC().Add(-72 * time.Hour)
	messages, err := imap.FetchRecent(ctx, since)
	if err != nil {
		w.log.Error("imap fetch failed",
			zap.String("profile_id", profile.ID),
			zap.String("imap_user", profile.IMAPUser),
			zap.Error(err),
		)
		_ = w.profiles.UpdateIMAPHealth(ctx, profile.ID, false, err.Error())
		return
	}
	_ = w.profiles.UpdateIMAPHealth(ctx, profile.ID, true, "")

	for _, msg := range messages {
		if err := w.matcher.ProcessMessage(ctx, profile, msg); err != nil {
			w.log.Error("process email failed",
				zap.String("profile_id", profile.ID),
				zap.Error(err),
				zap.String("message_id", msg.MessageID),
			)
		}
	}
}

var _ services.EmailPoller = (*MultiProfileWorker)(nil)
