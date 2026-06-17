package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
)

type PaymentProfileRepository struct {
	db            *sql.DB
	encryptSecret string
}

func NewPaymentProfileRepository(db *sql.DB, encryptSecret string) *PaymentProfileRepository {
	return &PaymentProfileRepository{db: db, encryptSecret: encryptSecret}
}

const profileSelectCols = `
	id, name, upi_id, payee_name, bank_code, imap_host, imap_port, imap_user,
	imap_password, sender_filter, parser_type, is_active,
	imap_last_ok_at, COALESCE(imap_last_error, ''), imap_last_checked_at,
	created_at, updated_at
`

func (r *PaymentProfileRepository) List(ctx context.Context, activeOnly bool) ([]*models.PaymentProfile, error) {
	q := fmt.Sprintf(`SELECT %s FROM payment_profiles`, profileSelectCols)
	if activeOnly {
		q += ` WHERE is_active = 1`
	}
	q += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.PaymentProfile
	for rows.Next() {
		p, err := r.scanProfileRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PaymentProfileRepository) GetByID(ctx context.Context, id string) (*models.PaymentProfile, error) {
	q := fmt.Sprintf(`SELECT %s FROM payment_profiles WHERE id = ? LIMIT 1`, profileSelectCols)
	return r.scanProfile(r.db.QueryRowContext(ctx, q, id))
}

func (r *PaymentProfileRepository) Count(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM payment_profiles`).Scan(&n)
	return n, err
}

func (r *PaymentProfileRepository) Create(ctx context.Context, p *models.PaymentProfile) error {
	encPass, err := security.Encrypt(p.IMAPPassword, r.encryptSecret)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO payment_profiles (
			id, name, upi_id, payee_name, bank_code, imap_host, imap_port,
			imap_user, imap_password, sender_filter, parser_type, is_active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.UPIID, p.PayeeName, p.BankCode, p.IMAPHost, p.IMAPPort,
		p.IMAPUser, encPass, p.SenderFilter, p.ParserType, boolToTiny(p.IsActive))
	return err
}

func (r *PaymentProfileRepository) Update(ctx context.Context, p *models.PaymentProfile) error {
	encPass, err := security.Encrypt(p.IMAPPassword, r.encryptSecret)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE payment_profiles SET
			name = ?, upi_id = ?, payee_name = ?, bank_code = ?,
			imap_host = ?, imap_port = ?, imap_user = ?, imap_password = ?,
			sender_filter = ?, parser_type = ?, is_active = ?, updated_at = NOW(3)
		WHERE id = ?
	`, p.Name, p.UPIID, p.PayeeName, p.BankCode, p.IMAPHost, p.IMAPPort,
		p.IMAPUser, encPass, p.SenderFilter, p.ParserType, boolToTiny(p.IsActive), p.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PaymentProfileRepository) UpdatePasswordOnly(ctx context.Context, id, plainPassword string) error {
	encPass, err := security.Encrypt(plainPassword, r.encryptSecret)
	if err != nil {
		return err
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE payment_profiles SET imap_password = ?, updated_at = NOW(3) WHERE id = ?
	`, encPass, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PaymentProfileRepository) scanProfile(row *sql.Row) (*models.PaymentProfile, error) {
	return r.scanProfileRow(row)
}

func (r *PaymentProfileRepository) scanProfileRow(scanner interface {
	Scan(dest ...any) error
}) (*models.PaymentProfile, error) {
	p := &models.PaymentProfile{}
	var encPass string
	var active int
	var lastOK, lastChecked sql.NullTime
	err := scanner.Scan(
		&p.ID, &p.Name, &p.UPIID, &p.PayeeName, &p.BankCode, &p.IMAPHost, &p.IMAPPort,
		&p.IMAPUser, &encPass, &p.SenderFilter, &p.ParserType, &active,
		&lastOK, &p.IMAPLastError, &lastChecked,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan profile: %w", err)
	}
	p.IsActive = active == 1
	if lastOK.Valid {
		p.IMAPLastOKAt = &lastOK.Time
	}
	if lastChecked.Valid {
		p.IMAPLastCheckedAt = &lastChecked.Time
	}
	plain, err := security.Decrypt(encPass, r.encryptSecret)
	if err != nil {
		// legacy plain-text password from bootstrap
		p.IMAPPassword = encPass
	} else {
		p.IMAPPassword = plain
	}
	return p, nil
}

func (r *PaymentProfileRepository) UpdateIMAPHealth(ctx context.Context, id string, ok bool, errMsg string) error {
	now := time.Now().UTC()
	if ok {
		_, err := r.db.ExecContext(ctx, `
			UPDATE payment_profiles SET
				imap_last_ok_at = ?, imap_last_error = NULL, imap_last_checked_at = ?, updated_at = NOW(3)
			WHERE id = ?
		`, now, now, id)
		return err
	}
	msg := errMsg
	if len(msg) > 500 {
		msg = msg[:500]
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE payment_profiles SET
			imap_last_error = ?, imap_last_checked_at = ?, updated_at = NOW(3)
		WHERE id = ?
	`, msg, now, id)
	return err
}

func (r *PaymentProfileRepository) ListIMAPAlerts(ctx context.Context, since time.Time) ([]*models.PaymentProfile, error) {
	q := fmt.Sprintf(`SELECT %s FROM payment_profiles
		WHERE is_active = 1 AND imap_last_ok_at IS NOT NULL AND imap_last_ok_at < ?
		   OR (is_active = 1 AND imap_last_ok_at IS NULL AND imap_last_checked_at IS NOT NULL AND imap_last_checked_at < ?)
		ORDER BY imap_last_ok_at ASC`, profileSelectCols)
	rows, err := r.db.QueryContext(ctx, q, since, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*models.PaymentProfile
	for rows.Next() {
		p, err := r.scanProfileRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func boolToTiny(v bool) int {
	if v {
		return 1
	}
	return 0
}

type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) GetByEmail(ctx context.Context, email string) (*models.AdminUser, error) {
	const q = `
		SELECT id, email, password_hash, COALESCE(name, ''), role, created_at
		FROM admin_users WHERE email = ? LIMIT 1
	`
	u := &models.AdminUser{}
	err := r.db.QueryRowContext(ctx, q, strings.ToLower(strings.TrimSpace(email))).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *MerchantRepository) GetByID(ctx context.Context, id string) (*models.Merchant, error) {
	const q = `
		SELECT id, name, domain, api_key, api_secret, webhook_url,
		       COALESCE(return_url, ''), status, COALESCE(payment_profile_id, ''),
		       created_at, updated_at
		FROM merchants WHERE id = ? LIMIT 1
	`
	m := &models.Merchant{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&m.ID, &m.Name, &m.Domain, &m.APIKey, &m.APISecret,
		&m.WebhookURL, &m.ReturnURL, &m.Status, &m.PaymentProfileID,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get merchant: %w", err)
	}
	return m, nil
}

func (r *MerchantRepository) List(ctx context.Context) ([]*models.Merchant, error) {
	const q = `
		SELECT id, name, domain, api_key, api_secret, webhook_url,
		       COALESCE(return_url, ''), status, COALESCE(payment_profile_id, ''),
		       created_at, updated_at
		FROM merchants ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.Merchant
	for rows.Next() {
		m := &models.Merchant{}
		if err := rows.Scan(
			&m.ID, &m.Name, &m.Domain, &m.APIKey, &m.APISecret,
			&m.WebhookURL, &m.ReturnURL, &m.Status, &m.PaymentProfileID,
			&m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *MerchantRepository) SetPaymentProfile(ctx context.Context, merchantID, profileID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE merchants SET payment_profile_id = ?, updated_at = NOW(3) WHERE id = ?
	`, nullStringPtr(profileID), merchantID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
