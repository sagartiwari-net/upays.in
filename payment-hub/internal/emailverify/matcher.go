package emailverify

import (
	"context"
	"strings"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/emailverify/parser"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/services"
	"go.uber.org/zap"
)

type Matcher struct {
	orders   *repository.OrderRepository
	bankTxns *repository.BankTxnRepository
	notifier *services.MerchantNotifier
	log      *zap.Logger
}

func NewMatcher(
	orders *repository.OrderRepository,
	bankTxns *repository.BankTxnRepository,
	notifier *services.MerchantNotifier,
	log *zap.Logger,
) *Matcher {
	return &Matcher{
		orders:   orders,
		bankTxns: bankTxns,
		notifier: notifier,
		log:      log,
	}
}

func (m *Matcher) ProcessMessage(ctx context.Context, profile *models.PaymentProfile, msg EmailMessage) error {
	alert, ok := parser.Parse(profile.ParserType, msg.Body)
	if !ok {
		return nil
	}

	exists, err := m.bankTxns.UTRExists(ctx, alert.UTR)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	excerpt := truncate(msg.Body, 500)
	order, err := m.findPendingOrder(ctx, profile.ID, alert)
	if err != nil {
		return err
	}

	orderID := ""
	if order != nil {
		orderID = order.ID
		if err := m.orders.MarkSuccess(ctx, order.ID, alert.UTR, nil); err != nil {
			return err
		}
		updated, err := m.orders.GetByHubOrderID(ctx, order.HubOrderID)
		if err == nil {
			m.notifier.NotifyAsync(updated, "payment.success")
			m.log.Info("payment matched",
				zap.String("hub_order_id", order.HubOrderID),
				zap.String("profile_id", profile.ID),
				zap.String("utr", alert.UTR),
				zap.Float64("amount", alert.Amount),
			)
		}
	} else {
		m.log.Warn("unmatched bank credit",
			zap.String("profile_id", profile.ID),
			zap.String("utr", alert.UTR),
			zap.Float64("amount", alert.Amount),
		)
	}

	return m.bankTxns.Record(ctx, alert.UTR, msg.MessageID, alert.Amount, orderID, profile.ID, excerpt)
}

func (m *Matcher) findPendingOrder(ctx context.Context, profileID string, alert *parser.CreditAlert) (*models.Order, error) {
	if o, err := m.orders.GetPendingByCustomerUTR(ctx, profileID, alert.UTR); err == nil {
		return o, nil
	} else if err != repository.ErrNotFound {
		return nil, err
	}

	o, err := m.orders.GetPendingByPayAmount(ctx, profileID, alert.Amount)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return o, nil
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}
