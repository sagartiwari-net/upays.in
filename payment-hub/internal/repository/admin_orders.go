package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
)

type OrderListItem struct {
	ID              string
	HubOrderID      string
	MerchantOrderID string
	MerchantID      string
	MerchantName    string
	MerchantDomain  string
	Amount          float64
	PayAmount       float64
	Currency        string
	Status          string
	CustomerEmail   string
	ProductName     string
	CustomerUTR     string
	PaidAt          *string
	CreatedAt       string
}

type OrderListFilter struct {
	Status     string
	MerchantID string
	Search     string
	Limit      int
	Offset     int
}

type DashboardStats struct {
	TodayOrders   int64
	TodaySuccess  int64
	TodayRevenue  float64
	TotalOrders   int64
	TotalSuccess  int64
	TotalRevenue  float64
	PendingOrders int64
}

func (r *OrderRepository) ListAdmin(ctx context.Context, f OrderListFilter) ([]OrderListItem, int64, error) {
	if f.Limit <= 0 || f.Limit > 100 {
		f.Limit = 25
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	where := []string{"1=1"}
	args := []interface{}{}

	if f.Status != "" {
		where = append(where, "o.status = ?")
		args = append(args, f.Status)
	}
	if f.MerchantID != "" {
		where = append(where, "o.merchant_id = ?")
		args = append(args, f.MerchantID)
	}
	if f.Search != "" {
		where = append(where, "(o.hub_order_id LIKE ? OR o.merchant_order_id LIKE ? OR o.customer_email LIKE ?)")
		like := "%" + f.Search + "%"
		args = append(args, like, like, like)
	}

	whereSQL := strings.Join(where, " AND ")

	var total int64
	countQ := fmt.Sprintf(`
		SELECT COUNT(*) FROM orders o
		JOIN merchants m ON m.id = o.merchant_id
		WHERE %s`, whereSQL)
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := fmt.Sprintf(`
		SELECT o.id, o.hub_order_id, o.merchant_order_id, o.merchant_id,
		       m.name, m.domain, o.amount, o.pay_amount, o.currency, o.status,
		       COALESCE(o.customer_email, ''), COALESCE(o.product_name, ''),
		       COALESCE(o.customer_utr, ''),
		       DATE_FORMAT(o.paid_at, '%%Y-%%m-%%dT%%H:%%i:%%sZ'),
		       DATE_FORMAT(o.created_at, '%%Y-%%m-%%dT%%H:%%i:%%sZ')
		FROM orders o
		JOIN merchants m ON m.id = o.merchant_id
		WHERE %s
		ORDER BY o.created_at DESC
		LIMIT ? OFFSET ?`, whereSQL)

	args = append(args, f.Limit, f.Offset)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []OrderListItem
	for rows.Next() {
		var item OrderListItem
		var paidAt *string
		if err := rows.Scan(
			&item.ID, &item.HubOrderID, &item.MerchantOrderID, &item.MerchantID,
			&item.MerchantName, &item.MerchantDomain, &item.Amount, &item.PayAmount,
			&item.Currency, &item.Status, &item.CustomerEmail, &item.ProductName,
			&item.CustomerUTR, &paidAt, &item.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		item.PaidAt = paidAt
		out = append(out, item)
	}
	return out, total, rows.Err()
}

func (r *OrderRepository) DashboardStats(ctx context.Context) (*DashboardStats, error) {
	const q = `
		SELECT
			COALESCE(SUM(CASE WHEN DATE(created_at) = CURDATE() THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN DATE(created_at) = CURDATE() AND status = 'success' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN DATE(created_at) = CURDATE() AND status = 'success' THEN amount ELSE 0 END), 0),
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'success' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0)
		FROM orders
	`
	s := &DashboardStats{}
	err := r.db.QueryRowContext(ctx, q).Scan(
		&s.TodayOrders, &s.TodaySuccess, &s.TodayRevenue,
		&s.TotalOrders, &s.TotalSuccess, &s.TotalRevenue, &s.PendingOrders,
	)
	return s, err
}

func (r *OrderRepository) DashboardStatsForMerchant(ctx context.Context, merchantID string) (*DashboardStats, error) {
	const q = `
		SELECT
			COALESCE(SUM(CASE WHEN DATE(created_at) = CURDATE() THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN DATE(created_at) = CURDATE() AND status = 'success' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN DATE(created_at) = CURDATE() AND status = 'success' THEN amount ELSE 0 END), 0),
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'success' THEN amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0)
		FROM orders WHERE merchant_id = ?
	`
	s := &DashboardStats{}
	err := r.db.QueryRowContext(ctx, q, merchantID).Scan(
		&s.TodayOrders, &s.TodaySuccess, &s.TodayRevenue,
		&s.TotalOrders, &s.TotalSuccess, &s.TotalRevenue, &s.PendingOrders,
	)
	return s, err
}

type MerchantRevenueRow struct {
	MerchantID   string
	MerchantName string
	Domain       string
	Orders       int64
	Revenue      float64
}

func (r *OrderRepository) MerchantRevenue(ctx context.Context, days int) ([]MerchantRevenueRow, error) {
	if days <= 0 {
		days = 30
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT m.id, m.name, m.domain,
		       COUNT(o.id),
		       COALESCE(SUM(CASE WHEN o.status = 'success' THEN o.amount ELSE 0 END), 0)
		FROM merchants m
		LEFT JOIN orders o ON o.merchant_id = m.id AND o.created_at >= DATE_SUB(NOW(), INTERVAL ? DAY)
		GROUP BY m.id, m.name, m.domain
		ORDER BY COALESCE(SUM(CASE WHEN o.status = 'success' THEN o.amount ELSE 0 END), 0) DESC
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MerchantRevenueRow
	for rows.Next() {
		var row MerchantRevenueRow
		if err := rows.Scan(&row.MerchantID, &row.MerchantName, &row.Domain, &row.Orders, &row.Revenue); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (*models.Order, error) {
	q := fmt.Sprintf(`SELECT %s FROM orders WHERE id = ? LIMIT 1`, orderSelectColumns)
	return r.scanOrder(r.db.QueryRowContext(ctx, q, id))
}
