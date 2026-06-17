package services

import (
	"context"
	"errors"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
)

var (
	ErrPlanLimitExceeded = repository.ErrPlanLimitExceeded
	ErrPlanExpired       = repository.ErrPlanExpired
	ErrNoSubscription    = repository.ErrNoSubscription
)

type SubscriptionUsage struct {
	PlanID       string    `json:"plan_id"`
	PlanName     string    `json:"plan_name"`
	PlanSlug     string    `json:"plan_slug"`
	PlanPrice    float64   `json:"plan_price_inr"`
	Status       string    `json:"status"`
	OrdersUsed   int       `json:"orders_used"`
	OrderLimit   int       `json:"order_limit"`
	StartsAt     time.Time `json:"starts_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	DaysLeft     int       `json:"days_left"`
	UsagePercent float64   `json:"usage_percent"`
	IsTrial      bool      `json:"is_trial"`
}

type SubscriptionService struct {
	subs *repository.SubscriptionRepository
}

func NewSubscriptionService(subs *repository.SubscriptionRepository) *SubscriptionService {
	return &SubscriptionService{subs: subs}
}

func (s *SubscriptionService) ActivatePlan(ctx context.Context, merchantID, planID, adminEmail, notes string) error {
	return s.subs.AssignPlan(ctx, merchantID, planID, adminEmail, notes)
}

func (s *SubscriptionService) ConsumeOrder(ctx context.Context, merchantID string) error {
	return s.subs.TryConsumeOrder(ctx, merchantID)
}

func (s *SubscriptionService) ReleaseOrder(ctx context.Context, merchantID string) error {
	return s.subs.ReleaseOrderSlot(ctx, merchantID)
}

func (s *SubscriptionService) GetUsage(ctx context.Context, merchantID string) (*SubscriptionUsage, error) {
	sub, err := s.subs.GetActiveForMerchant(ctx, merchantID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNoSubscription
		}
		return nil, err
	}
	return usageFromSub(sub), nil
}

func (s *SubscriptionService) ListPlans(ctx context.Context) ([]models.SubscriptionPlan, error) {
	return s.subs.ListPlans(ctx, true)
}

func (s *SubscriptionService) ListAllPlans(ctx context.Context) ([]models.SubscriptionPlan, error) {
	return s.subs.ListPlans(ctx, false)
}

func (s *SubscriptionService) CreatePlan(ctx context.Context, in repository.PlanInput) (*models.SubscriptionPlan, error) {
	if in.IsRecommended {
		_ = s.subs.ClearRecommendedExcept(ctx, "")
	}
	return s.subs.CreatePlan(ctx, in)
}

func (s *SubscriptionService) UpdatePlan(ctx context.Context, id string, in repository.PlanInput) (*models.SubscriptionPlan, error) {
	if in.IsRecommended {
		_ = s.subs.ClearRecommendedExcept(ctx, id)
	}
	return s.subs.UpdatePlan(ctx, id, in)
}

func usageFromSub(sub *models.MerchantSubscription) *SubscriptionUsage {
	limit := sub.OrderLimit
	if limit <= 0 {
		limit = 1
	}
	pct := float64(sub.OrdersUsed) / float64(limit) * 100
	daysLeft := int(time.Until(sub.ExpiresAt).Hours() / 24)
	if daysLeft < 0 {
		daysLeft = 0
	}
	return &SubscriptionUsage{
		PlanID:       sub.PlanID,
		PlanName:     sub.PlanName,
		PlanSlug:     sub.PlanSlug,
		PlanPrice:    sub.PlanPrice,
		Status:       sub.Status,
		OrdersUsed:   sub.OrdersUsed,
		OrderLimit:   sub.OrderLimit,
		StartsAt:     sub.StartsAt,
		ExpiresAt:    sub.ExpiresAt,
		DaysLeft:     daysLeft,
		UsagePercent: pct,
		IsTrial:      sub.PlanSlug == "trial",
	}
}
