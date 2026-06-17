package handlers

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/assets/amember"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/assets/plugins"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/emailverify/parser"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/services"
)

type MerchantPortalHandler struct {
	auth      *services.MerchantAuthService
	users     *repository.MerchantUserRepository
	merchants *repository.MerchantRepository
	orders    *repository.OrderRepository
	orderSvc  *services.OrderService
	profiles  *services.ProfileService
	subs      *services.SubscriptionService
}

func NewMerchantPortalHandler(
	auth *services.MerchantAuthService,
	users *repository.MerchantUserRepository,
	merchants *repository.MerchantRepository,
	orders *repository.OrderRepository,
	orderSvc *services.OrderService,
	profiles *services.ProfileService,
	subs *services.SubscriptionService,
) *MerchantPortalHandler {
	return &MerchantPortalHandler{
		auth: auth, users: users, merchants: merchants, orders: orders,
		orderSvc: orderSvc, profiles: profiles, subs: subs,
	}
}

func (h *MerchantPortalHandler) Register(c *fiber.Ctx) error {
	var body struct {
		Email        string `json:"email"`
		Password     string `json:"password"`
		Name         string `json:"name"`
		BusinessName string `json:"business_name"`
		Domain       string `json:"domain"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	token, user, merchant, err := h.auth.Register(c.Context(), services.RegisterInput{
		Email: body.Email, Password: body.Password, Name: body.Name,
		BusinessName: body.BusinessName, Domain: body.Domain,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrEmailTaken):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "email already registered"})
		case errors.Is(err, services.ErrDomainTaken):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "domain already registered"})
		case errors.Is(err, services.ErrWeakPassword):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"token":    token,
		"user":     merchantUserResponse(user),
		"merchant": merchantPortalResponse(merchant, true),
	})
}

func (h *MerchantPortalHandler) Login(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	token, user, err := h.auth.Login(c.Context(), body.Email, body.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid email or password"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "login failed"})
	}
	m, _ := h.merchants.GetByID(c.Context(), user.MerchantID)
	return c.JSON(fiber.Map{
		"token":    token,
		"user":     merchantUserResponse(user),
		"merchant": merchantPortalResponse(m, false),
	})
}

func (h *MerchantPortalHandler) Me(c *fiber.Ctx) error {
	userID := c.Locals("merchant_user_id").(string)
	merchantID := c.Locals("merchant_id").(string)
	user, err := h.users.GetByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}
	m, err := h.merchants.GetByID(c.Context(), merchantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "merchant not found"})
	}
	resp := fiber.Map{
		"user":     merchantUserResponse(user),
		"merchant": merchantPortalResponse(m, false),
	}
	if m.PaymentProfileID != "" {
		if p, err := h.profiles.Get(c.Context(), m.PaymentProfileID); err == nil {
			resp["payment_profile"] = services.ProfileResponse(p)
		}
	}
	return c.JSON(resp)
}

func (h *MerchantPortalHandler) Dashboard(c *fiber.Ctx) error {
	merchantID := c.Locals("merchant_id").(string)
	stats, err := h.orders.DashboardStatsForMerchant(c.Context(), merchantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load stats"})
	}
	successRate := 0.0
	if stats.TotalOrders > 0 {
		successRate = float64(stats.TotalSuccess) / float64(stats.TotalOrders) * 100
	}
	return c.JSON(fiber.Map{
		"today_orders":   stats.TodayOrders,
		"today_success":  stats.TodaySuccess,
		"today_revenue":  stats.TodayRevenue,
		"total_orders":   stats.TotalOrders,
		"total_revenue":  stats.TotalRevenue,
		"pending_orders": stats.PendingOrders,
		"success_rate":   successRate,
	})
}

func (h *MerchantPortalHandler) Subscription(c *fiber.Ctx) error {
	merchantID := c.Locals("merchant_id").(string)
	usage, err := h.subs.GetUsage(c.Context(), merchantID)
	if err != nil {
		if errors.Is(err, services.ErrNoSubscription) {
			return c.JSON(fiber.Map{"subscription": nil})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load subscription"})
	}
	return c.JSON(fiber.Map{"subscription": usage})
}

func (h *MerchantPortalHandler) ListPlans(c *fiber.Ctx) error {
	plans, err := h.subs.ListPlans(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list plans"})
	}
	out := make([]fiber.Map, 0, len(plans))
	for _, p := range plans {
		if p.Slug == "trial" {
			continue
		}
		out = append(out, fiber.Map{
			"id": p.ID, "slug": p.Slug, "name": p.Name, "price_inr": p.PriceINR,
			"validity_days": p.ValidityDays, "order_limit": p.OrderLimit,
			"is_recommended": p.IsRecommended, "features_json": p.FeaturesJSON,
		})
	}
	return c.JSON(fiber.Map{"plans": out})
}

func (h *MerchantPortalHandler) ListOrders(c *fiber.Ctx) error {
	merchantID := c.Locals("merchant_id").(string)
	f := repository.OrderListFilter{
		MerchantID: merchantID,
		Status:     c.Query("status"),
		Search:     c.Query("q"),
		Limit:      50,
	}
	items, total, err := h.orders.ListAdmin(c.Context(), f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list orders"})
	}
	return c.JSON(fiber.Map{"orders": items, "total": total})
}

func (h *MerchantPortalHandler) UpdateMerchant(c *fiber.Ctx) error {
	merchantID := c.Locals("merchant_id").(string)
	var body struct {
		Name       string `json:"name"`
		Domain     string `json:"domain"`
		WebhookURL string `json:"webhook_url"`
		ReturnURL  string `json:"return_url"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	m, err := h.merchants.GetByID(c.Context(), merchantID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = m.Name
	}
	domain := strings.TrimSpace(body.Domain)
	if domain == "" {
		domain = m.Domain
	}
	if err := h.merchants.Update(c.Context(), merchantID, repository.MerchantInput{
		Name: name, Domain: domain,
		WebhookURL: body.WebhookURL, ReturnURL: body.ReturnURL, Status: m.Status,
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	m, _ = h.merchants.GetByID(c.Context(), merchantID)
	return c.JSON(merchantPortalResponse(m, false))
}

func (h *MerchantPortalHandler) RegenerateSecret(c *fiber.Ctx) error {
	merchantID := c.Locals("merchant_id").(string)
	secret, err := h.merchants.RegenerateAPISecret(c.Context(), merchantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	m, _ := h.merchants.GetByID(c.Context(), merchantID)
	return c.JSON(fiber.Map{
		"api_key":    m.APIKey,
		"api_secret": secret,
		"message":    "New secret generated. Update your website integration.",
	})
}

func (h *MerchantPortalHandler) SetupPaymentProfile(c *fiber.Ctx) error {
	merchantID := c.Locals("merchant_id").(string)
	userID := c.Locals("merchant_user_id").(string)

	var in services.ProfileInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if in.Name == "" || in.UPIID == "" || in.IMAPUser == "" || in.IMAPPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "profile name, upi_id, imap_user, imap_password required",
		})
	}

	m, err := h.merchants.GetByID(c.Context(), merchantID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "merchant not found"})
	}

	var p *models.PaymentProfile
	if m.PaymentProfileID != "" {
		p, err = h.profiles.Update(c.Context(), m.PaymentProfileID, in)
	} else {
		p, err = h.profiles.Create(c.Context(), in)
		if err == nil {
			_ = h.profiles.AssignMerchant(c.Context(), merchantID, p.ID)
		}
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	_ = h.users.SetOnboardingDone(c.Context(), userID, true)
	return c.JSON(fiber.Map{
		"payment_profile": services.ProfileResponse(p),
		"onboarding_done":   true,
	})
}

func (h *MerchantPortalHandler) CompleteOnboarding(c *fiber.Ctx) error {
	userID := c.Locals("merchant_user_id").(string)
	if err := h.users.SetOnboardingDone(c.Context(), userID, true); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"onboarding_done": true})
}

func (h *MerchantPortalHandler) ParserTypes(c *fiber.Ctx) error {
	types := make([]fiber.Map, 0, len(parser.AllParserTypes()))
	for _, p := range parser.AllParserTypes() {
		types = append(types, fiber.Map{
			"id": p.ID, "label": p.Label, "sender_filter": p.SenderFilter, "bank_code": p.BankCode,
		})
	}
	return c.JSON(fiber.Map{"parser_types": types	})
}

func (h *MerchantPortalHandler) DownloadAmemberPlugin(c *fiber.Ctx) error {
	content := amember.Plugin
	if len(content) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "plugin not found"})
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fw, _ := zw.Create("upipays.php")
	_, _ = fw.Write(content)
	_ = zw.Close()
	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", `attachment; filename="upipays-amember-plugin.zip"`)
	return c.Send(buf.Bytes())
}

func (h *MerchantPortalHandler) DownloadWooCommercePlugin(c *fiber.Ctx) error {
	dir := plugins.FindWooCommercePluginDir()
	if dir == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "plugin not found"})
	}
	buf, err := plugins.ZipDirectory(dir)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "zip failed"})
	}
	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", `attachment; filename="upipays-woocommerce.zip"`)
	return c.Send(buf)
}

func merchantUserResponse(u *models.MerchantUser) fiber.Map {
	if u == nil {
		return nil
	}
	return fiber.Map{
		"id":              u.ID,
		"email":           u.Email,
		"name":            u.Name,
		"merchant_id":     u.MerchantID,
		"onboarding_done": u.OnboardingDone,
	}
}

func merchantPortalResponse(m *models.Merchant, includeSecret bool) fiber.Map {
	if m == nil {
		return nil
	}
	resp := fiber.Map{
		"id":                 m.ID,
		"name":               m.Name,
		"domain":             m.Domain,
		"api_key":            m.APIKey,
		"webhook_url":        m.WebhookURL,
		"return_url":         m.ReturnURL,
		"status":             m.Status,
		"payment_profile_id": m.PaymentProfileID,
		"hub_url":            "https://upays.in",
	}
	if includeSecret {
		resp["api_secret"] = m.APISecret
	}
	return resp
}

func (h *MerchantPortalHandler) CreatePaymentLink(c *fiber.Ctx) error {
	merchantID := c.Locals("merchant_id").(string)
	var body struct {
		Amount      float64 `json:"amount"`
		ProductName string  `json:"product_name"`
		ReturnURL   string  `json:"return_url"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "amount must be greater than zero"})
	}
	if strings.TrimSpace(body.ProductName) == "" {
		body.ProductName = "Payment"
	}

	m, err := h.merchants.GetByID(c.Context(), merchantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "merchant not found"})
	}

	returnURL := strings.TrimSpace(body.ReturnURL)
	if returnURL == "" {
		returnURL = m.ReturnURL
	}
	if returnURL == "" {
		returnURL = "https://upays.in"
	}

	orderID := fmt.Sprintf("LINK-%s", time.Now().UTC().Format("20060102150405"))
	result, err := h.orderSvc.Create(c.Context(), m, models.CreateOrderInput{
		OrderID:  orderID,
		Amount:   body.Amount,
		Currency: "INR",
		Customer: models.CustomerInput{Name: "Customer", Email: "customer@example.com"},
		Product:  models.ProductInput{Name: body.ProductName},
		ReturnURL: returnURL,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrPlanLimitExceeded), errors.Is(err, services.ErrPlanExpired), errors.Is(err, services.ErrNoSubscription):
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, services.ErrInvalidInput):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create payment link"})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"order_id":    orderID,
		"hub_order_id": result.HubOrderID,
		"payment_url": result.PaymentURL,
		"expires_at":  result.ExpiresAt.Format(time.RFC3339),
	})
}
