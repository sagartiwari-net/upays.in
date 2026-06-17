package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
)

type MerchantUserRepository struct {
	db *sql.DB
}

func NewMerchantUserRepository(db *sql.DB) *MerchantUserRepository {
	return &MerchantUserRepository{db: db}
}

func (r *MerchantUserRepository) GetByEmail(ctx context.Context, email string) (*models.MerchantUser, error) {
	const q = `
		SELECT id, email, password_hash, name, merchant_id, onboarding_done, created_at, updated_at
		FROM merchant_users WHERE email = ? LIMIT 1
	`
	u := &models.MerchantUser{}
	var done int
	err := r.db.QueryRowContext(ctx, q, strings.ToLower(strings.TrimSpace(email))).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.MerchantID, &done, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.OnboardingDone = done == 1
	return u, nil
}

func (r *MerchantUserRepository) GetByID(ctx context.Context, id string) (*models.MerchantUser, error) {
	const q = `
		SELECT id, email, password_hash, name, merchant_id, onboarding_done, created_at, updated_at
		FROM merchant_users WHERE id = ? LIMIT 1
	`
	u := &models.MerchantUser{}
	var done int
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.MerchantID, &done, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	u.OnboardingDone = done == 1
	return u, nil
}

func (r *MerchantUserRepository) Create(ctx context.Context, u *models.MerchantUser) error {
	return r.CreateTx(ctx, r.db, u)
}

func (r *MerchantUserRepository) CreateTx(ctx context.Context, exec interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}, u *models.MerchantUser) error {
	done := 0
	if u.OnboardingDone {
		done = 1
	}
	const q = `
		INSERT INTO merchant_users (id, email, password_hash, name, merchant_id, onboarding_done)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, q, u.ID, strings.ToLower(u.Email), u.PasswordHash, u.Name, u.MerchantID, done)
	if err != nil {
		if isDuplicateKey(err) {
			return ErrDuplicateOrder
		}
		return fmt.Errorf("create merchant user: %w", err)
	}
	return nil
}

func (r *MerchantUserRepository) SetOnboardingDone(ctx context.Context, userID string, done bool) error {
	v := 0
	if done {
		v = 1
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE merchant_users SET onboarding_done = ?, updated_at = NOW(3) WHERE id = ?
	`, v, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
