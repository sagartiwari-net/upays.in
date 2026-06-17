package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/order"
)

func (r *OrderRepository) GetByPaymentToken(ctx context.Context, token string) (*models.Order, error) {
	q := fmt.Sprintf(`SELECT %s FROM orders WHERE payment_token = ? LIMIT 1`, orderSelectColumns)
	return r.scanOrder(r.db.QueryRowContext(ctx, q, token))
}

func (r *OrderRepository) GetByHubOrderID(ctx context.Context, hubOrderID string) (*models.Order, error) {
	q := fmt.Sprintf(`SELECT %s FROM orders WHERE hub_order_id = ? LIMIT 1`, orderSelectColumns)
	return r.scanOrder(r.db.QueryRowContext(ctx, q, hubOrderID))
}

func (r *OrderRepository) TransitionStatus(ctx context.Context, orderID, fromStatus, toStatus string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders SET status = ?, updated_at = NOW(3)
		WHERE id = ? AND status = ?
	`, toStatus, orderID, fromStatus)
	if err != nil {
		return fmt.Errorf("transition status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *OrderRepository) MarkSuccess(ctx context.Context, orderID, phonepeTxnID string, phonepeResponse []byte) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = ?, phonepe_txn_id = ?, phonepe_response = ?, paid_at = NOW(3), updated_at = NOW(3)
		WHERE id = ? AND status IN (?, ?)
	`, models.OrderStatusSuccess, phonepeTxnID, nullJSON(phonepeResponse), orderID,
		models.OrderStatusPending, models.OrderStatusProcessing)
	if err != nil {
		return fmt.Errorf("mark success: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// idempotent: already success
		var status string
		err := r.db.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = ?`, orderID).Scan(&status)
		if err == nil && status == models.OrderStatusSuccess {
			return nil
		}
		return ErrNotFound
	}
	return nil
}

func (r *OrderRepository) MarkFailed(ctx context.Context, orderID string, phonepeResponse []byte) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders
		SET status = ?, phonepe_response = ?, updated_at = NOW(3)
		WHERE id = ? AND status IN (?, ?)
	`, models.OrderStatusFailed, nullJSON(phonepeResponse), orderID,
		models.OrderStatusPending, models.OrderStatusProcessing)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		var status string
		err := r.db.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = ?`, orderID).Scan(&status)
		if err == nil && order.IsFinalStatus(status) {
			return nil
		}
		return ErrNotFound
	}
	return nil
}

func (r *OrderRepository) SavePhonePeResponse(ctx context.Context, orderID string, phonepeResponse []byte) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE orders SET phonepe_response = ?, updated_at = NOW(3) WHERE id = ?
	`, nullJSON(phonepeResponse), orderID)
	return err
}

func (r *OrderRepository) scanOrder(row *sql.Row) (*models.Order, error) {
	return r.scanOrderFields(row)
}

func nullJSON(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	return b
}

func (r *OrderRepository) ExpirePendingOrders(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		UPDATE orders SET status = ?, updated_at = NOW(3)
		WHERE status = ? AND expires_at < ?
	`, models.OrderStatusFailed, models.OrderStatusPending, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
