package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/config"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
	"go.uber.org/zap"
)

type ProfileService struct {
	profiles  *repository.PaymentProfileRepository
	merchants *repository.MerchantRepository
	cfg       *config.Config
	log       *zap.Logger
}

func NewProfileService(
	profiles *repository.PaymentProfileRepository,
	merchants *repository.MerchantRepository,
	cfg *config.Config,
	log *zap.Logger,
) *ProfileService {
	return &ProfileService{profiles: profiles, merchants: merchants, cfg: cfg, log: log}
}

func (s *ProfileService) BootstrapFromEnv(ctx context.Context) error {
	if !s.cfg.ProfileBootstrap {
		s.log.Info("profile bootstrap disabled via PROFILE_BOOTSTRAP=false")
		return s.ensureMerchantsAssigned(ctx)
	}
	count, err := s.profiles.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		if err := s.syncIMAPFromEnv(ctx); err != nil {
			s.log.Warn("sync imap from env", zap.Error(err))
		}
		return s.ensureMerchantsAssigned(ctx)
	}
	if s.cfg.UPIID == "" {
		s.log.Warn("no payment profiles and UPI_ID empty — skipping bootstrap")
		return nil
	}

	profile := &models.PaymentProfile{
		ID:           security.NewID(),
		Name:         "Default UPI",
		UPIID:        s.cfg.UPIID,
		PayeeName:    s.cfg.UPIPayeeName,
		BankCode:     "hdfc",
		IMAPHost:     s.cfg.IMAPHost,
		IMAPPort:     s.cfg.IMAPPort,
		IMAPUser:     s.cfg.IMAPUser,
		IMAPPassword: s.cfg.IMAPPassword,
		SenderFilter: s.cfg.IMAPSenderFilter,
		ParserType:   "hdfc",
		IsActive:     true,
	}
	if err := s.profiles.Create(ctx, profile); err != nil {
		return fmt.Errorf("bootstrap profile: %w", err)
	}
	s.log.Info("bootstrapped default payment profile from .env", zap.String("profile_id", profile.ID))
	return s.assignAllMerchants(ctx, profile.ID)
}

func (s *ProfileService) ensureMerchantsAssigned(ctx context.Context) error {
	profiles, err := s.profiles.List(ctx, true)
	if err != nil || len(profiles) == 0 {
		return err
	}
	defaultProfile := profiles[0]
	merchants, err := s.merchants.List(ctx)
	if err != nil {
		return err
	}
	for _, m := range merchants {
		if m.PaymentProfileID == "" {
			if err := s.merchants.SetPaymentProfile(ctx, m.ID, defaultProfile.ID); err != nil {
				s.log.Warn("assign default profile", zap.String("merchant", m.Domain), zap.Error(err))
			}
		}
	}
	return nil
}

func (s *ProfileService) syncIMAPFromEnv(ctx context.Context) error {
	if s.cfg.IMAPUser == "" || s.cfg.IMAPPassword == "" {
		return nil
	}
	profiles, err := s.profiles.List(ctx, true)
	if err != nil {
		return err
	}
	for _, p := range profiles {
		p.IMAPHost = s.cfg.IMAPHost
		if s.cfg.IMAPPort > 0 {
			p.IMAPPort = s.cfg.IMAPPort
		}
		p.IMAPUser = s.cfg.IMAPUser
		p.IMAPPassword = s.cfg.IMAPPassword
		if s.cfg.IMAPSenderFilter != "" {
			p.SenderFilter = s.cfg.IMAPSenderFilter
		}
		if err := s.profiles.Update(ctx, p); err != nil {
			return fmt.Errorf("update profile %s: %w", p.ID, err)
		}
	}
	if len(profiles) > 0 {
		s.log.Info("synced IMAP settings from .env", zap.Int("profiles", len(profiles)))
	}
	return nil
}

func (s *ProfileService) assignAllMerchants(ctx context.Context, profileID string) error {
	merchants, err := s.merchants.List(ctx)
	if err != nil {
		return err
	}
	for _, m := range merchants {
		if err := s.merchants.SetPaymentProfile(ctx, m.ID, profileID); err != nil {
			return err
		}
	}
	return nil
}

func (s *ProfileService) ResolveForMerchant(ctx context.Context, merchant *models.Merchant) (*models.PaymentProfile, error) {
	if merchant.PaymentProfileID != "" {
		return s.profiles.GetByID(ctx, merchant.PaymentProfileID)
	}
	profiles, err := s.profiles.List(ctx, true)
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, errors.New("no active payment profile configured")
	}
	return profiles[0], nil
}

func (s *ProfileService) ResolveForOrder(ctx context.Context, order *models.Order) (*models.PaymentProfile, error) {
	if order.PaymentProfileID != "" {
		return s.profiles.GetByID(ctx, order.PaymentProfileID)
	}
	merchant, err := s.merchants.GetByID(ctx, order.MerchantID)
	if err != nil {
		return nil, err
	}
	return s.ResolveForMerchant(ctx, merchant)
}

func (s *ProfileService) List(ctx context.Context) ([]*models.PaymentProfile, error) {
	return s.profiles.List(ctx, false)
}

func (s *ProfileService) Get(ctx context.Context, id string) (*models.PaymentProfile, error) {
	return s.profiles.GetByID(ctx, id)
}

func (s *ProfileService) Create(ctx context.Context, in ProfileInput) (*models.PaymentProfile, error) {
	p := in.toModel(security.NewID())
	if err := s.profiles.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProfileService) Update(ctx context.Context, id string, in ProfileInput) (*models.PaymentProfile, error) {
	existing, err := s.profiles.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	p := in.toModel(id)
	if in.IMAPPassword == "" {
		p.IMAPPassword = existing.IMAPPassword
	}
	if err := s.profiles.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProfileService) AssignMerchant(ctx context.Context, merchantID, profileID string) error {
	if _, err := s.profiles.GetByID(ctx, profileID); err != nil {
		return err
	}
	return s.merchants.SetPaymentProfile(ctx, merchantID, profileID)
}

type ProfileInput struct {
	Name           string `json:"name"`
	UPIID          string `json:"upi_id"`
	PayeeName      string `json:"payee_name"`
	BankCode       string `json:"bank_code"`
	IMAPHost       string `json:"imap_host"`
	IMAPPort       int    `json:"imap_port"`
	IMAPUser       string `json:"imap_user"`
	IMAPPassword   string `json:"imap_password"`
	SenderFilter   string `json:"sender_filter"`
	ParserType     string `json:"parser_type"`
	IsActive       bool   `json:"is_active"`
}

func (in ProfileInput) toModel(id string) *models.PaymentProfile {
	p := &models.PaymentProfile{
		ID:           id,
		Name:         in.Name,
		UPIID:        in.UPIID,
		PayeeName:    in.PayeeName,
		BankCode:     in.BankCode,
		IMAPHost:     in.IMAPHost,
		IMAPPort:     in.IMAPPort,
		IMAPUser:     in.IMAPUser,
		IMAPPassword: in.IMAPPassword,
		SenderFilter: in.SenderFilter,
		ParserType:   in.ParserType,
		IsActive:     in.IsActive,
	}
	if p.PayeeName == "" {
		p.PayeeName = "UPIPays"
	}
	if p.BankCode == "" {
		p.BankCode = "hdfc"
	}
	if p.IMAPHost == "" {
		p.IMAPHost = "imap.gmail.com"
	}
	if p.IMAPPort == 0 {
		p.IMAPPort = 993
	}
	if p.SenderFilter == "" {
		p.SenderFilter = "hdfcbank"
	}
	if p.ParserType == "" {
		p.ParserType = "hdfc"
	}
	return p
}

func ProfileResponse(p *models.PaymentProfile) map[string]interface{} {
	return map[string]interface{}{
		"id":            p.ID,
		"name":          p.Name,
		"upi_id":        p.UPIID,
		"payee_name":    p.PayeeName,
		"bank_code":     p.BankCode,
		"imap_host":     p.IMAPHost,
		"imap_port":     p.IMAPPort,
		"imap_user":     p.IMAPUser,
		"imap_password": security.MaskSecret(p.IMAPPassword),
		"sender_filter": p.SenderFilter,
		"parser_type":   p.ParserType,
		"is_active":     p.IsActive,
		"imap_last_ok_at":      formatTimePtr(p.IMAPLastOKAt),
		"imap_last_error":      p.IMAPLastError,
		"imap_last_checked_at": formatTimePtr(p.IMAPLastCheckedAt),
		"created_at":    p.CreatedAt,
		"updated_at":    p.UpdatedAt,
	}
}

func formatTimePtr(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
