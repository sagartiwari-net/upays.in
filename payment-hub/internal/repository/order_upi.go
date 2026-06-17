package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
)

const orderSelectColumns = `
	id, hub_order_id, merchant_id, merchant_order_id, payment_token,
	amount, pay_amount, currency, payment_provider, COALESCE(payment_profile_id, ''), status,
	COALESCE(customer_email, ''), COALESCE(customer_name, ''), COALESCE(customer_phone, ''),
	COALESCE(product_name, ''), COALESCE(product_description, ''),
	return_url, COALESCE(webhook_url, ''), COALESCE(phonepe_txn_id, ''), COALESCE(customer_utr, ''),
	paid_at, expires_at, created_at, updated_at
`

type BankTxnRepository struct {
	db *sql.DB
}

func NewBankTxnRepository(db *sql.DB) *BankTxnRepository {
	return &BankTxnRepository{db: db}
}

func (r *BankTxnRepository) UTRExists(ctx context.Context, utr string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM processed_bank_txns WHERE utr = ?`, utr).Scan(&n)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *BankTxnRepository) Record(ctx context.Context, utr, messageID string, amount float64, orderID, profileID, excerpt string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO processed_bank_txns (id, utr, email_message_id, amount, order_id, payment_profile_id, raw_excerpt)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, security.NewID(), utr, messageID, amount, nullStringPtr(orderID), nullStringPtr(profileID), nullString(excerpt))
	if err != nil && isDuplicateKey(err) {
		return nil
	}
	return err
}

type UnmatchedTxn struct {
	ID           string
	UTR          string
	Amount       float64
	ProfileID    string
	ProfileName  string
	RawExcerpt   string
	CreatedAt    string
}

func (r *BankTxnRepository) ListUnmatched(ctx context.Context, limit, offset int) ([]UnmatchedTxn, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM processed_bank_txns WHERE order_id IS NULL
	`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT bt.id, bt.utr, bt.amount, COALESCE(bt.payment_profile_id, ''),
		       COALESCE(pp.name, ''), COALESCE(bt.raw_excerpt, ''),
		       DATE_FORMAT(bt.created_at, '%%Y-%%m-%%dT%%H:%%i:%%sZ')
		FROM processed_bank_txns bt
		LEFT JOIN payment_profiles pp ON pp.id = bt.payment_profile_id
		WHERE bt.order_id IS NULL
		ORDER BY bt.created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []UnmatchedTxn
	for rows.Next() {
		var item UnmatchedTxn
		if err := rows.Scan(&item.ID, &item.UTR, &item.Amount, &item.ProfileID, &item.ProfileName, &item.RawExcerpt, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (r *BankTxnRepository) LinkOrder(ctx context.Context, txnID, orderID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE processed_bank_txns SET order_id = ? WHERE id = ? AND order_id IS NULL
	`, orderID, txnID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *BankTxnRepository) GetByID(ctx context.Context, id string) (utr string, amount float64, orderID sql.NullString, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT utr, amount, order_id FROM processed_bank_txns WHERE id = ?
	`, id).Scan(&utr, &amount, &orderID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, sql.NullString{}, ErrNotFound
	}
	return
}

func nullStringPtr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (r *OrderRepository) PendingPayAmountExists(ctx context.Context, profileID string, payAmount float64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM orders
		WHERE status = ? AND pay_amount = ? AND payment_profile_id = ?
	`, models.OrderStatusPending, payAmount, profileID).Scan(&n)
	return n > 0, err
}

func (r *OrderRepository) GetPendingByPayAmount(ctx context.Context, profileID string, payAmount float64) (*models.Order, error) {
	q := fmt.Sprintf(`SELECT %s FROM orders WHERE status = ? AND pay_amount = ? AND payment_profile_id = ? ORDER BY created_at ASC LIMIT 1`, orderSelectColumns)
	return r.scanOrder(r.db.QueryRowContext(ctx, q, models.OrderStatusPending, payAmount, profileID))
}

func (r *OrderRepository) GetPendingByCustomerUTR(ctx context.Context, profileID, utr string) (*models.Order, error) {
	q := fmt.Sprintf(`SELECT %s FROM orders WHERE status = ? AND customer_utr = ? AND payment_profile_id = ? LIMIT 1`, orderSelectColumns)
	return r.scanOrder(r.db.QueryRowContext(ctx, q, models.OrderStatusPending, utr, profileID))
}

func (r *OrderRepository) SetCustomerUTR(ctx context.Context, orderID, utr string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders SET customer_utr = ?, updated_at = NOW(3)
		WHERE id = ? AND status = ?
	`, utr, orderID, models.OrderStatusPending)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func AmountsEqual(a, b float64) bool {
	return int64(math.Round(a*100)) == int64(math.Round(b*100))
}

func (r *OrderRepository) scanOrderFields(row interface {
	Scan(dest ...any) error
}) (*models.Order, error) {
	o := &models.Order{}
	var paidAt sql.NullTime
	err := row.Scan(
		&o.ID, &o.HubOrderID, &o.MerchantID, &o.MerchantOrderID, &o.PaymentToken,
		&o.Amount, &o.PayAmount, &o.Currency, &o.PaymentProvider, &o.PaymentProfileID, &o.Status,
		&o.CustomerEmail, &o.CustomerName, &o.CustomerPhone,
		&o.ProductName, &o.ProductDescription,
		&o.ReturnURL, &o.WebhookURL, &o.PhonePeTxnID, &o.CustomerUTR,
		&paidAt, &o.ExpiresAt, &o.CreatedAt, &o.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan order: %w", err)
	}
	if paidAt.Valid {
		o.PaidAt = &paidAt.Time
	}
	if o.PayAmount == 0 {
		o.PayAmount = o.Amount
	}
	return o, nil
}
