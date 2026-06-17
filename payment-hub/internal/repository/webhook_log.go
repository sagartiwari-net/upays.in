package repository

import (
	"context"
	"database/sql"
)

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

type WebhookLogRepo struct {
	db *sql.DB
}

func NewWebhookLogRepository(db *sql.DB) *WebhookLogRepo {
	return &WebhookLogRepo{db: db}
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
