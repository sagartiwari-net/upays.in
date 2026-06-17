package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
)

const (
	WebhookStatusPending   = "pending"
	WebhookStatusDelivered = "delivered"
	WebhookStatusRetry     = "retry"
	WebhookStatusFailed    = "failed"
)

type WebhookLogRecord struct {
	ID          string
	OrderID     string
	MerchantID  string
	Direction   string
	Payload     string
	Status      string
	RetryCount  int
	NextRetryAt *time.Time
}

type WebhookLogRepo struct {
	db *sql.DB
}

func NewWebhookLogRepository(db *sql.DB) *WebhookLogRepo {
	return &WebhookLogRepo{db: db}
}

func (r *WebhookLogRepo) CreateOutbound(ctx context.Context, orderID, merchantID, payload string) (string, error) {
	id := security.NewID()
	const q = `
		INSERT INTO webhook_logs (id, order_id, merchant_id, direction, payload, status)
		VALUES (?, ?, ?, 'outbound', ?, ?)
	`
	_, err := r.db.ExecContext(ctx, q, id, orderID, merchantID, payload, WebhookStatusPending)
	return id, err
}

func (r *WebhookLogRepo) MarkDelivered(ctx context.Context, id string, responseCode int, responseBody string) error {
	const q = `
		UPDATE webhook_logs
		SET status = ?, response_code = ?, response_body = ?, next_retry_at = NULL
		WHERE id = ?
	`
	var body interface{}
	if responseBody != "" {
		body = responseBody
	}
	_, err := r.db.ExecContext(ctx, q, WebhookStatusDelivered, responseCode, body, id)
	return err
}

func (r *WebhookLogRepo) ScheduleRetry(ctx context.Context, id string, responseCode *int, responseBody string, retryCount int, nextRetry time.Time) error {
	status := WebhookStatusRetry
	if retryCount >= maxWebhookRetries {
		status = WebhookStatusFailed
		nextRetry = time.Time{}
	}
	const q = `
		UPDATE webhook_logs
		SET status = ?, response_code = ?, response_body = ?, retry_count = ?, next_retry_at = ?
		WHERE id = ?
	`
	var code interface{}
	if responseCode != nil {
		code = *responseCode
	}
	var next interface{}
	if !nextRetry.IsZero() {
		next = nextRetry
	}
	_, err := r.db.ExecContext(ctx, q, status, code, responseBody, retryCount, next, id)
	return err
}

const maxWebhookRetries = 5

func WebhookRetryDelay(retryCount int) time.Duration {
	delays := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		15 * time.Minute,
		60 * time.Minute,
		4 * time.Hour,
	}
	if retryCount < 0 {
		retryCount = 0
	}
	if retryCount >= len(delays) {
		return delays[len(delays)-1]
	}
	return delays[retryCount]
}

func (r *WebhookLogRepo) ListDueRetries(ctx context.Context, limit int) ([]WebhookLogRecord, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	const q = `
		SELECT id, order_id, merchant_id, direction, payload, status, retry_count, next_retry_at
		FROM webhook_logs
		WHERE status = ? AND next_retry_at IS NOT NULL AND next_retry_at <= UTC_TIMESTAMP(3)
		ORDER BY next_retry_at ASC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, q, WebhookStatusRetry, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []WebhookLogRecord
	for rows.Next() {
		var rec WebhookLogRecord
		var next sql.NullTime
		if err := rows.Scan(
			&rec.ID, &rec.OrderID, &rec.MerchantID, &rec.Direction, &rec.Payload,
			&rec.Status, &rec.RetryCount, &next,
		); err != nil {
			return nil, err
		}
		if next.Valid {
			rec.NextRetryAt = &next.Time
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *WebhookLogRepo) GetByID(ctx context.Context, id string) (*WebhookLogRecord, error) {
	const q = `
		SELECT id, order_id, merchant_id, direction, payload, status, retry_count, next_retry_at
		FROM webhook_logs WHERE id = ? LIMIT 1
	`
	rec := &WebhookLogRecord{}
	var next sql.NullTime
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&rec.ID, &rec.OrderID, &rec.MerchantID, &rec.Direction, &rec.Payload,
		&rec.Status, &rec.RetryCount, &next,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if next.Valid {
		rec.NextRetryAt = &next.Time
	}
	return rec, nil
}

type WebhookLogItem struct {
	ID           string
	OrderID      string
	HubOrderID   string
	MerchantName string
	Direction    string
	Status       string
	ResponseCode *int
	CreatedAt    string
}

func (r *WebhookLogRepo) List(ctx context.Context, limit, offset int) ([]WebhookLogItem, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_logs`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT wl.id, wl.order_id, COALESCE(o.hub_order_id, ''), COALESCE(m.name, ''),
		       wl.direction, wl.status, wl.response_code,
		       DATE_FORMAT(wl.created_at, '%%Y-%%m-%%dT%%H:%%i:%%sZ')
		FROM webhook_logs wl
		LEFT JOIN orders o ON o.id = wl.order_id
		LEFT JOIN merchants m ON m.id = wl.merchant_id
		ORDER BY wl.created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []WebhookLogItem
	for rows.Next() {
		var item WebhookLogItem
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.HubOrderID, &item.MerchantName,
			&item.Direction, &item.Status, &item.ResponseCode, &item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, item)
	}
	return out, total, rows.Err()
}
