package services

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailTaken       = errors.New("email already registered")
	ErrDomainTaken      = errors.New("domain already registered")
	ErrWeakPassword     = errors.New("password must be at least 8 characters")
)

type MerchantAuthService struct {
	db        *sql.DB
	users     *repository.MerchantUserRepository
	merchants *repository.MerchantRepository
	subs      *repository.SubscriptionRepository
	jwtSecret string
}

func NewMerchantAuthService(
	db *sql.DB,
	users *repository.MerchantUserRepository,
	merchants *repository.MerchantRepository,
	subs *repository.SubscriptionRepository,
	jwtSecret string,
) *MerchantAuthService {
	return &MerchantAuthService{db: db, users: users, merchants: merchants, subs: subs, jwtSecret: jwtSecret}
}

type RegisterInput struct {
	Email        string
	Password     string
	Name         string
	BusinessName string
	Domain       string
}

func (s *MerchantAuthService) Register(ctx context.Context, in RegisterInput) (string, *models.MerchantUser, *models.Merchant, error) {
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Domain = strings.TrimSpace(strings.ToLower(in.Domain))
	in.BusinessName = strings.TrimSpace(in.BusinessName)
	in.Name = strings.TrimSpace(in.Name)

	if in.Email == "" || in.Password == "" || in.BusinessName == "" || in.Domain == "" {
		return "", nil, nil, errors.New("email, password, business name and domain required")
	}
	if len(in.Password) < 8 {
		return "", nil, nil, ErrWeakPassword
	}
	if _, err := s.users.GetByEmail(ctx, in.Email); err == nil {
		return "", nil, nil, ErrEmailTaken
	} else if !errors.Is(err, repository.ErrNotFound) {
		return "", nil, nil, err
	}

	hash, err := HashPassword(in.Password)
	if err != nil {
		return "", nil, nil, err
	}

	slug := merchantSlugFromDomain(in.Domain)
	merchantID := security.NewID()
	userID := security.NewID()

	m := &models.Merchant{
		ID:         merchantID,
		Name:       in.BusinessName,
		Domain:     in.Domain,
		APIKey:     security.NewAPIKey("mk_" + slug),
		APISecret:  "sk_" + security.NewAPISecret(),
		WebhookURL: "",
		ReturnURL:  "",
		Status:     models.MerchantStatusActive,
	}
	u := &models.MerchantUser{
		ID:           userID,
		Email:        in.Email,
		PasswordHash: hash,
		Name:         in.Name,
		MerchantID:   merchantID,
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", nil, nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	if err := s.merchants.CreateTx(ctx, tx, m); err != nil {
		if errors.Is(err, repository.ErrDuplicateOrder) {
			return "", nil, nil, ErrDomainTaken
		}
		return "", nil, nil, err
	}
	if err := s.users.CreateTx(ctx, tx, u); err != nil {
		if errors.Is(err, repository.ErrDuplicateOrder) {
			return "", nil, nil, ErrEmailTaken
		}
		return "", nil, nil, err
	}
	if s.subs != nil {
		if err := s.subs.AssignTrialTx(ctx, tx, merchantID); err != nil {
			return "", nil, nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", nil, nil, err
	}

	token, err := security.IssueMerchantToken(u.ID, m.ID, u.Email, s.jwtSecret, 7*24*time.Hour)
	if err != nil {
		return "", nil, nil, err
	}
	return token, u, m, nil
}

func (s *MerchantAuthService) Login(ctx context.Context, email, password string) (string, *models.MerchantUser, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}
	token, err := security.IssueMerchantToken(user.ID, user.MerchantID, user.Email, s.jwtSecret, 7*24*time.Hour)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func merchantSlugFromDomain(domain string) string {
	slug := strings.ReplaceAll(strings.Split(domain, ".")[0], "-", "")
	if slug == "" {
		return "merchant"
	}
	if len(slug) > 12 {
		slug = slug[:12]
	}
	return slug
}
