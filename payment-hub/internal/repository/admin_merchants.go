package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
)

type MerchantInput struct {
	Name       string
	Domain     string
	WebhookURL string
	ReturnURL  string
	Status     string
}

func (r *MerchantRepository) Create(ctx context.Context, m *models.Merchant) error {
	return r.CreateTx(ctx, r.db, m)
}

func (r *MerchantRepository) CreateTx(ctx context.Context, exec sqlExecutor, m *models.Merchant) error {
	const q = `
		INSERT INTO merchants (id, name, domain, api_key, api_secret, webhook_url, return_url, status, payment_profile_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := exec.ExecContext(ctx, q,
		m.ID, m.Name, m.Domain, m.APIKey, m.APISecret,
		m.WebhookURL, nullString(m.ReturnURL), m.Status, nullStringPtr(m.PaymentProfileID),
	)
	if err != nil {
		if isDuplicateKey(err) {
			return ErrDuplicateOrder
		}
		return fmt.Errorf("create merchant: %w", err)
	}
	return nil
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func (r *MerchantRepository) Update(ctx context.Context, id string, in MerchantInput) error {
	status := in.Status
	if status == "" {
		status = models.MerchantStatusActive
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE merchants
		SET name = ?, domain = ?, webhook_url = ?, return_url = ?, status = ?, updated_at = NOW(3)
		WHERE id = ?
	`, in.Name, in.Domain, in.WebhookURL, nullString(in.ReturnURL), status, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MerchantRepository) SetStatus(ctx context.Context, id, status string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE merchants SET status = ?, updated_at = NOW(3) WHERE id = ?
	`, status, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MerchantRepository) RegenerateAPISecret(ctx context.Context, id string) (string, error) {
	secret := "sk_" + security.NewAPISecret()
	res, err := r.db.ExecContext(ctx, `
		UPDATE merchants SET api_secret = ?, updated_at = NOW(3) WHERE id = ?
	`, secret, id)
	if err != nil {
		return "", err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return "", ErrNotFound
	}
	return secret, nil
}
