package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
)

var (
	ErrPlanLimitExceeded = errors.New("plan order limit exceeded")
	ErrPlanExpired       = errors.New("subscription expired")
	ErrNoSubscription    = errors.New("no active subscription")
)

type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) ListPlans(ctx context.Context, activeOnly bool) ([]models.SubscriptionPlan, error) {
	q := `
		SELECT id, slug, name, price_inr, validity_days, order_limit, is_recommended, sort_order, is_active, features_json, created_at, updated_at
		FROM subscription_plans
	`
	if activeOnly {
		q += ` WHERE is_active = 1`
	}
	q += ` ORDER BY sort_order ASC, price_inr ASC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.SubscriptionPlan
	for rows.Next() {
		var p models.SubscriptionPlan
		var rec, active int
		var features sql.NullString
		if err := rows.Scan(
			&p.ID, &p.Slug, &p.Name, &p.PriceINR, &p.ValidityDays, &p.OrderLimit,
			&rec, &p.SortOrder, &active, &features, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.IsRecommended = rec == 1
		p.IsActive = active == 1
		if features.Valid {
			p.FeaturesJSON = features.String
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *SubscriptionRepository) GetPlanByID(ctx context.Context, id string) (*models.SubscriptionPlan, error) {
	const q = `
		SELECT id, slug, name, price_inr, validity_days, order_limit, is_recommended, sort_order, is_active, features_json, created_at, updated_at
		FROM subscription_plans WHERE id = ? LIMIT 1
	`
	p := &models.SubscriptionPlan{}
	var rec, active int
	var features sql.NullString
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&p.ID, &p.Slug, &p.Name, &p.PriceINR, &p.ValidityDays, &p.OrderLimit,
		&rec, &p.SortOrder, &active, &features, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.IsRecommended = rec == 1
	p.IsActive = active == 1
	if features.Valid {
		p.FeaturesJSON = features.String
	}
	return p, nil
}

func (r *SubscriptionRepository) GetPlanBySlug(ctx context.Context, slug string) (*models.SubscriptionPlan, error) {
	const q = `
		SELECT id, slug, name, price_inr, validity_days, order_limit, is_recommended, sort_order, is_active, features_json, created_at, updated_at
		FROM subscription_plans WHERE slug = ? LIMIT 1
	`
	p := &models.SubscriptionPlan{}
	var rec, active int
	var features sql.NullString
	err := r.db.QueryRowContext(ctx, q, slug).Scan(
		&p.ID, &p.Slug, &p.Name, &p.PriceINR, &p.ValidityDays, &p.OrderLimit,
		&rec, &p.SortOrder, &active, &features, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.IsRecommended = rec == 1
	p.IsActive = active == 1
	if features.Valid {
		p.FeaturesJSON = features.String
	}
	return p, nil
}

func (r *SubscriptionRepository) ExpireStale(ctx context.Context, merchantID string) error {
	const q = `
		UPDATE merchant_subscriptions
		SET status = ?
		WHERE merchant_id = ? AND status = ? AND expires_at <= UTC_TIMESTAMP(3)
	`
	_, err := r.db.ExecContext(ctx, q, models.SubStatusExpired, merchantID, models.SubStatusActive)
	return err
}

func (r *SubscriptionRepository) GetActiveForMerchant(ctx context.Context, merchantID string) (*models.MerchantSubscription, error) {
	if err := r.ExpireStale(ctx, merchantID); err != nil {
		return nil, err
	}
	const q = `
		SELECT ms.id, ms.merchant_id, ms.plan_id, ms.status, ms.starts_at, ms.expires_at,
		       ms.orders_used, ms.order_limit, ms.activated_by, ms.notes, ms.created_at, ms.updated_at,
		       sp.name, sp.slug, sp.price_inr
		FROM merchant_subscriptions ms
		JOIN subscription_plans sp ON sp.id = ms.plan_id
		WHERE ms.merchant_id = ? AND ms.status = ?
		ORDER BY ms.created_at DESC
		LIMIT 1
	`
	s := &models.MerchantSubscription{}
	var activatedBy, notes sql.NullString
	err := r.db.QueryRowContext(ctx, q, merchantID, models.SubStatusActive).Scan(
		&s.ID, &s.MerchantID, &s.PlanID, &s.Status, &s.StartsAt, &s.ExpiresAt,
		&s.OrdersUsed, &s.OrderLimit, &activatedBy, &notes, &s.CreatedAt, &s.UpdatedAt,
		&s.PlanName, &s.PlanSlug, &s.PlanPrice,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if activatedBy.Valid {
		s.ActivatedBy = activatedBy.String
	}
	if notes.Valid {
		s.Notes = notes.String
	}
	if time.Now().UTC().After(s.ExpiresAt) {
		_, _ = r.db.ExecContext(ctx,
			`UPDATE merchant_subscriptions SET status = ? WHERE id = ?`,
			models.SubStatusExpired, s.ID,
		)
		return nil, ErrNotFound
	}
	return s, nil
}

func (r *SubscriptionRepository) CreateSubscription(ctx context.Context, s *models.MerchantSubscription) error {
	return r.CreateSubscriptionTx(ctx, r.db, s)
}

func (r *SubscriptionRepository) CreateSubscriptionTx(ctx context.Context, exec interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}, s *models.MerchantSubscription) error {
	const q = `
		INSERT INTO merchant_subscriptions
		(id, merchant_id, plan_id, status, starts_at, expires_at, orders_used, order_limit, activated_by, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	var activatedBy, notes interface{}
	if s.ActivatedBy != "" {
		activatedBy = s.ActivatedBy
	}
	if s.Notes != "" {
		notes = s.Notes
	}
	_, err := exec.ExecContext(ctx, q,
		s.ID, s.MerchantID, s.PlanID, s.Status, s.StartsAt, s.ExpiresAt,
		s.OrdersUsed, s.OrderLimit, activatedBy, notes,
	)
	return err
}

func (r *SubscriptionRepository) ExpireActiveForMerchantTx(ctx context.Context, exec interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}, merchantID string) error {
	const q = `
		UPDATE merchant_subscriptions SET status = ?
		WHERE merchant_id = ? AND status = ?
	`
	_, err := exec.ExecContext(ctx, q, models.SubStatusExpired, merchantID, models.SubStatusActive)
	return err
}

func (r *SubscriptionRepository) AssignPlan(ctx context.Context, merchantID, planID, activatedBy, notes string) error {
	plan, err := r.GetPlanByID(ctx, planID)
	if err != nil {
		return err
	}
	if !plan.IsActive {
		return errors.New("plan is inactive")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := r.ExpireActiveForMerchantTx(ctx, tx, merchantID); err != nil {
		return err
	}

	now := time.Now().UTC()
	sub := &models.MerchantSubscription{
		ID:         security.NewID(),
		MerchantID: merchantID,
		PlanID:     plan.ID,
		Status:     models.SubStatusActive,
		StartsAt:   now,
		ExpiresAt:  now.Add(time.Duration(plan.ValidityDays) * 24 * time.Hour),
		OrdersUsed: 0,
		OrderLimit: plan.OrderLimit,
		ActivatedBy: activatedBy,
		Notes:       notes,
	}
	if err := r.CreateSubscriptionTx(ctx, tx, sub); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SubscriptionRepository) AssignTrialTx(ctx context.Context, exec interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}, merchantID string) error {
	plan, err := r.GetPlanBySlug(ctx, "trial")
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	sub := &models.MerchantSubscription{
		ID:         security.NewID(),
		MerchantID: merchantID,
		PlanID:     plan.ID,
		Status:     models.SubStatusActive,
		StartsAt:   now,
		ExpiresAt:  now.Add(time.Duration(plan.ValidityDays) * 24 * time.Hour),
		OrdersUsed: 0,
		OrderLimit: plan.OrderLimit,
		Notes:      "auto trial on signup",
	}
	return r.CreateSubscriptionTx(ctx, exec, sub)
}

// TryConsumeOrder atomically increments usage if within plan limits.
func (r *SubscriptionRepository) TryConsumeOrder(ctx context.Context, merchantID string) error {
	if err := r.ExpireStale(ctx, merchantID); err != nil {
		return err
	}
	const q = `
		UPDATE merchant_subscriptions
		SET orders_used = orders_used + 1
		WHERE merchant_id = ? AND status = ? AND expires_at > UTC_TIMESTAMP(3) AND orders_used < order_limit
		ORDER BY created_at DESC
		LIMIT 1
	`
	res, err := r.db.ExecContext(ctx, q, merchantID, models.SubStatusActive)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		sub, err := r.GetActiveForMerchant(ctx, merchantID)
		if errors.Is(err, ErrNotFound) {
			return ErrPlanExpired
		}
		if err != nil {
			return err
		}
		if sub.OrdersUsed >= sub.OrderLimit {
			return ErrPlanLimitExceeded
		}
		return ErrPlanExpired
	}
	return nil
}

func (r *SubscriptionRepository) ReleaseOrderSlot(ctx context.Context, merchantID string) error {
	const q = `
		UPDATE merchant_subscriptions
		SET orders_used = GREATEST(orders_used - 1, 0)
		WHERE merchant_id = ? AND status = ?
		ORDER BY created_at DESC
		LIMIT 1
	`
	_, err := r.db.ExecContext(ctx, q, merchantID, models.SubStatusActive)
	return err
}
