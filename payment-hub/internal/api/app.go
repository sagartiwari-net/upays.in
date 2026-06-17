package api

import (
	"context"
	"database/sql"
	"time"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/api/handlers"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/api/middleware"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/config"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/emailverify"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/payment"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/phonepe"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"
)

type AppServices struct {
	EmailWorker *emailverify.MultiProfileWorker
}

func NewApp(cfg *config.Config, log *zap.Logger, db *sql.DB) (*fiber.App, *AppServices) {
	app := fiber.New(fiber.Config{
		AppName:      "UPIPays",
		ServerHeader: "UPIPays",
	})

	app.Use(recover.New())
	app.Use(middleware.NormalizePath())
	app.Use(middleware.RequestLogger(log))

	merchantRepo := repository.NewMerchantRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	bankTxnRepo := repository.NewBankTxnRepository(db)
	profileRepo := repository.NewPaymentProfileRepository(db, cfg.EncryptSecret())
	adminRepo := repository.NewAdminRepository(db)
	notifier := services.NewMerchantNotifier(merchantRepo, log)

	profileService := services.NewProfileService(profileRepo, merchantRepo, cfg, log)
	if err := profileService.BootstrapFromEnv(context.Background()); err != nil {
		log.Warn("profile bootstrap failed", zap.Error(err))
	}

	orderService := services.NewOrderService(orderRepo, profileService, cfg.AppURL, cfg.OrderExpiryMinutes, cfg.PaymentProvider)

	var paymentService *services.PaymentService
	var upiService *services.UPIService
	var emailWorker *emailverify.MultiProfileWorker

	if cfg.PaymentProvider == payment.ProviderPhonePe {
		ppClient := phonepe.NewClient(cfg.PhonePeClientConfig())
		paymentService = services.NewPaymentService(orderRepo, merchantRepo, ppClient, notifier, cfg.AppURL, log)
	} else {
		matcher := emailverify.NewMatcher(orderRepo, bankTxnRepo, notifier, log)
		interval := time.Duration(cfg.EmailPollIntervalSec) * time.Second
		emailWorker = emailverify.NewMultiProfileWorker(profileRepo, matcher, orderRepo, interval, log)
		upiService = services.NewUPIService(
			orderRepo, merchantRepo, profileService, notifier, cfg.AppURL, emailWorker, log,
		)
	}

	healthHandler := handlers.NewHealthHandler(db, cfg.AppEnv)
	orderHandler := handlers.NewOrderHandler(orderService, log)
	checkoutHandler := handlers.NewCheckoutHandler(cfg.PaymentProvider, upiService, paymentService)

	adminAuth := services.NewAdminAuthService(adminRepo, cfg.JWTSecret)
	webhookLogRepo := repository.NewWebhookLogRepository(db)
	adminHandler := handlers.NewAdminHandler(
		adminAuth, profileService, profileRepo, merchantRepo, orderRepo,
		bankTxnRepo, webhookLogRepo, notifier, emailWorker,
	)

	app.Get("/health", healthHandler.Health)

	app.Get("/pay/:token", checkoutHandler.Show)

	if cfg.PaymentProvider == payment.ProviderUPIEmail {
		app.Get("/pay/:token/qr", checkoutHandler.QR)
		app.Get("/pay/:token/status", checkoutHandler.Status)
		app.Post("/pay/:token/utr", checkoutHandler.SubmitUTR)
	} else {
		app.Post("/pay/:token/pay", checkoutHandler.Pay)
		app.All("/pay/:token/return", checkoutHandler.Return)
		webhookHandler := handlers.NewPhonePeWebhookHandler(paymentService)
		app.Post("/webhooks/phonepe", webhookHandler.Handle)
	}

	admin := app.Group("/admin/api")
	admin.Post("/auth/login", adminHandler.Login)

	adminProtected := admin.Group("", middleware.AdminAuth(cfg.JWTSecret))
	adminProtected.Get("/auth/me", adminHandler.Me)
	adminProtected.Get("/dashboard", adminHandler.Dashboard)
	adminProtected.Get("/orders/export", adminHandler.ExportOrdersCSV)
	adminProtected.Post("/orders/:id/manual-approve", adminHandler.ManualApproveOrder)
	adminProtected.Get("/unmatched", adminHandler.ListUnmatched)
	adminProtected.Get("/dashboard/merchant-revenue", adminHandler.MerchantRevenue)
	adminProtected.Get("/dashboard/imap-alerts", adminHandler.IMAPAlerts)
	adminProtected.Get("/downloads/amember-plugin", adminHandler.DownloadAmemberPlugin)
	adminProtected.Post("/payment-profiles/:id/test-imap", adminHandler.TestProfileIMAP)
	adminProtected.Post("/payment-profiles/:id/test-parse", adminHandler.TestProfileParse)
	adminProtected.Post("/payment-profiles/:id/trigger-poll", adminHandler.TriggerProfilePoll)
	adminProtected.Get("/orders", adminHandler.ListOrders)
	adminProtected.Get("/webhooks", adminHandler.ListWebhooks)
	adminProtected.Get("/payment-profiles", adminHandler.ListProfiles)
	adminProtected.Get("/payment-profiles/parser-types", adminHandler.ParserTypes)
	adminProtected.Get("/payment-profiles/:id", adminHandler.GetProfile)
	adminProtected.Post("/payment-profiles", adminHandler.CreateProfile)
	adminProtected.Put("/payment-profiles/:id", adminHandler.UpdateProfile)
	adminProtected.Get("/merchants", adminHandler.ListMerchants)
	adminProtected.Get("/merchants/:id", adminHandler.GetMerchant)
	adminProtected.Post("/merchants", adminHandler.CreateMerchant)
	adminProtected.Put("/merchants/:id", adminHandler.UpdateMerchant)
	adminProtected.Post("/merchants/:id/regenerate-secret", adminHandler.RegenerateMerchantSecret)
	adminProtected.Put("/merchants/:id/payment-profile", adminHandler.AssignMerchantProfile)
	adminProtected.Post("/onboarding/website", adminHandler.OnboardWebsite)

	registerAdminUI(app)
	registerPublicSite(app)

	maxSkew := time.Duration(cfg.SignatureMaxAgeMinutes) * time.Minute
	rateLimiter := middleware.NewRateLimiter(100, 20, time.Minute)

	api := app.Group("/api/v1")
	api.Use(rateLimiter.Middleware())
	api.Use(middleware.MerchantAuth(merchantRepo, maxSkew))

	api.Post("/orders/create", orderHandler.Create)
	api.Get("/orders/:order_id/verify", orderHandler.Verify)

	return app, &AppServices{EmailWorker: emailWorker}
}

func StartEmailWorker(ctx context.Context, cfg *config.Config, svc *AppServices) {
	if cfg.EmailWorkerEnabled() && svc != nil && svc.EmailWorker != nil {
		go svc.EmailWorker.Start(ctx)
	}
}
