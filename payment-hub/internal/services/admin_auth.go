package services

import (
	"context"
	"errors"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type AdminAuthService struct {
	admins    *repository.AdminRepository
	jwtSecret string
}

func NewAdminAuthService(admins *repository.AdminRepository, jwtSecret string) *AdminAuthService {
	return &AdminAuthService{admins: admins, jwtSecret: jwtSecret}
}

func (s *AdminAuthService) Login(ctx context.Context, email, password string) (string, *models.AdminUser, error) {
	user, err := s.admins.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}
	token, err := security.IssueAdminToken(user.ID, user.Email, user.Role, s.jwtSecret, 24*time.Hour)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}
