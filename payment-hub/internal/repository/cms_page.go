package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
)

type CMSPageRepository struct {
	db *sql.DB
}

func NewCMSPageRepository(db *sql.DB) *CMSPageRepository {
	return &CMSPageRepository{db: db}
}

func scanCMSPage(row interface {
	Scan(dest ...interface{}) error
}) (*models.CMSPage, error) {
	p := &models.CMSPage{}
	var meta, navLabel sql.NullString
	var showNav int
	err := row.Scan(
		&p.ID, &p.Slug, &p.Title, &meta, &p.BodyHTML, &p.Status,
		&showNav, &navLabel, &p.SortOrder, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if meta.Valid {
		p.MetaDescription = meta.String
	}
	if navLabel.Valid {
		p.NavLabel = navLabel.String
	}
	p.ShowInNav = showNav == 1
	return p, nil
}

const cmsSelectCols = `
	id, slug, title, meta_description, body_html, status, show_in_nav, nav_label, sort_order, created_at, updated_at
`

func (r *CMSPageRepository) List(ctx context.Context) ([]models.CMSPage, error) {
	q := `SELECT ` + cmsSelectCols + ` FROM cms_pages ORDER BY sort_order ASC, title ASC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.CMSPage
	for rows.Next() {
		p, err := scanCMSPage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *CMSPageRepository) ListPublishedNav(ctx context.Context) ([]models.CMSPage, error) {
	q := `
		SELECT ` + cmsSelectCols + `
		FROM cms_pages
		WHERE status = ? AND show_in_nav = 1
		ORDER BY sort_order ASC, title ASC
	`
	rows, err := r.db.QueryContext(ctx, q, models.CMSStatusPublished)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.CMSPage
	for rows.Next() {
		p, err := scanCMSPage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *CMSPageRepository) GetByID(ctx context.Context, id string) (*models.CMSPage, error) {
	q := `SELECT ` + cmsSelectCols + ` FROM cms_pages WHERE id = ? LIMIT 1`
	p, err := scanCMSPage(r.db.QueryRowContext(ctx, q, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *CMSPageRepository) GetPublishedBySlug(ctx context.Context, slug string) (*models.CMSPage, error) {
	q := `
		SELECT ` + cmsSelectCols + `
		FROM cms_pages WHERE slug = ? AND status = ? LIMIT 1
	`
	p, err := scanCMSPage(r.db.QueryRowContext(ctx, q, strings.ToLower(strings.TrimSpace(slug)), models.CMSStatusPublished))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

func (r *CMSPageRepository) GetBySlug(ctx context.Context, slug string) (*models.CMSPage, error) {
	q := `SELECT ` + cmsSelectCols + ` FROM cms_pages WHERE slug = ? LIMIT 1`
	p, err := scanCMSPage(r.db.QueryRowContext(ctx, q, strings.ToLower(strings.TrimSpace(slug))))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return p, err
}

type CMSPageInput struct {
	Slug            string
	Title           string
	MetaDescription string
	BodyHTML        string
	Status          string
	ShowInNav       bool
	NavLabel        string
	SortOrder       int
}

func (r *CMSPageRepository) Create(ctx context.Context, in CMSPageInput) (*models.CMSPage, error) {
	id := security.NewID()
	showNav := 0
	if in.ShowInNav {
		showNav = 1
	}
	status := in.Status
	if status == "" {
		status = models.CMSStatusDraft
	}
	const q = `
		INSERT INTO cms_pages
		(id, slug, title, meta_description, body_html, status, show_in_nav, nav_label, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	var meta, nav interface{}
	if in.MetaDescription != "" {
		meta = in.MetaDescription
	}
	navLabel := in.NavLabel
	if navLabel == "" {
		navLabel = in.Title
	}
	nav = navLabel
	_, err := r.db.ExecContext(ctx, q,
		id, strings.ToLower(strings.TrimSpace(in.Slug)), in.Title, meta, in.BodyHTML,
		status, showNav, nav, in.SortOrder,
	)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrDuplicateOrder
		}
		return nil, err
	}
	return r.GetByID(ctx, id)
}

func (r *CMSPageRepository) Update(ctx context.Context, id string, in CMSPageInput) (*models.CMSPage, error) {
	showNav := 0
	if in.ShowInNav {
		showNav = 1
	}
	status := in.Status
	if status == "" {
		status = models.CMSStatusDraft
	}
	const q = `
		UPDATE cms_pages SET
			slug = ?, title = ?, meta_description = ?, body_html = ?, status = ?,
			show_in_nav = ?, nav_label = ?, sort_order = ?
		WHERE id = ?
	`
	var meta interface{}
	if in.MetaDescription != "" {
		meta = in.MetaDescription
	}
	navLabel := in.NavLabel
	if navLabel == "" {
		navLabel = in.Title
	}
	res, err := r.db.ExecContext(ctx, q,
		strings.ToLower(strings.TrimSpace(in.Slug)), in.Title, meta, in.BodyHTML, status,
		showNav, navLabel, in.SortOrder, id,
	)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrDuplicateOrder
		}
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *CMSPageRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM cms_pages WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
