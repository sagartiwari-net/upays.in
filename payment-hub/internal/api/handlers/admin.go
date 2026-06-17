package handlers

import (
	"errors"
	"strconv"
	"strings"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/emailverify"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/emailverify/parser"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/services"
	"github.com/gofiber/fiber/v2"
)

type AdminHandler struct {
	auth        *services.AdminAuthService
	profiles    *services.ProfileService
	profileRepo *repository.PaymentProfileRepository
	merchants   *repository.MerchantRepository
	orders      *repository.OrderRepository
	bankTxns    *repository.BankTxnRepository
	webhooks    *repository.WebhookLogRepo
	notifier    *services.MerchantNotifier
	emailWorker emailverify.ProfilePoller
	subs        *services.SubscriptionService
	cms         *repository.CMSPageRepository
}

func NewAdminHandler(
	auth *services.AdminAuthService,
	profiles *services.ProfileService,
	profileRepo *repository.PaymentProfileRepository,
	merchants *repository.MerchantRepository,
	orders *repository.OrderRepository,
	bankTxns *repository.BankTxnRepository,
	webhooks *repository.WebhookLogRepo,
	notifier *services.MerchantNotifier,
	emailWorker emailverify.ProfilePoller,
	subs *services.SubscriptionService,
	cms *repository.CMSPageRepository,
) *AdminHandler {
	return &AdminHandler{
		auth: auth, profiles: profiles, profileRepo: profileRepo, merchants: merchants,
		orders: orders, bankTxns: bankTxns, webhooks: webhooks,
		notifier: notifier, emailWorker: emailWorker, subs: subs, cms: cms,
	}
}

func (h *AdminHandler) Login(c *fiber.Ctx) error {
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
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "login failed"})
	}
	return c.JSON(fiber.Map{
		"token": token,
		"admin": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}

func (h *AdminHandler) Me(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AdminHandler) Dashboard(c *fiber.Ctx) error {
	stats, err := h.orders.DashboardStats(c.Context())
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
		"total_success":  stats.TotalSuccess,
		"total_revenue":  stats.TotalRevenue,
		"pending_orders": stats.PendingOrders,
		"success_rate":   successRate,
	})
}

func (h *AdminHandler) ListOrders(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	items, total, err := h.orders.ListAdmin(c.Context(), repository.OrderListFilter{
		Status:     c.Query("status"),
		MerchantID: c.Query("merchant_id"),
		Search:     strings.TrimSpace(c.Query("q")),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list orders"})
	}
	out := make([]fiber.Map, 0, len(items))
	for _, o := range items {
		row := fiber.Map{
			"id":                o.ID,
			"hub_order_id":      o.HubOrderID,
			"merchant_order_id": o.MerchantOrderID,
			"merchant_id":       o.MerchantID,
			"merchant_name":     o.MerchantName,
			"merchant_domain":   o.MerchantDomain,
			"amount":            o.Amount,
			"pay_amount":        o.PayAmount,
			"currency":          o.Currency,
			"status":            o.Status,
			"customer_email":    o.CustomerEmail,
			"product_name":      o.ProductName,
			"customer_utr":      o.CustomerUTR,
			"created_at":        o.CreatedAt,
		}
		if o.PaidAt != nil {
			row["paid_at"] = *o.PaidAt
		}
		out = append(out, row)
	}
	return c.JSON(fiber.Map{"orders": out, "total": total})
}

func (h *AdminHandler) ListProfiles(c *fiber.Ctx) error {
	profiles, err := h.profiles.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list profiles"})
	}
	out := make([]map[string]interface{}, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, services.ProfileResponse(p))
	}
	return c.JSON(fiber.Map{"profiles": out})
}

func (h *AdminHandler) GetProfile(c *fiber.Ctx) error {
	p, err := h.profiles.Get(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get profile"})
	}
	return c.JSON(services.ProfileResponse(p))
}

func (h *AdminHandler) CreateProfile(c *fiber.Ctx) error {
	var in services.ProfileInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if in.Name == "" || in.UPIID == "" || in.IMAPUser == "" || in.IMAPPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name, upi_id, imap_user, imap_password required"})
	}
	p, err := h.profiles.Create(c.Context(), in)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(services.ProfileResponse(p))
}

func (h *AdminHandler) UpdateProfile(c *fiber.Ctx) error {
	var in services.ProfileInput
	if err := c.BodyParser(&in); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	p, err := h.profiles.Update(c.Context(), c.Params("id"), in)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(services.ProfileResponse(p))
}

func (h *AdminHandler) ListMerchants(c *fiber.Ctx) error {
	merchants, err := h.merchants.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list merchants"})
	}
	out := make([]fiber.Map, 0, len(merchants))
	for _, m := range merchants {
		out = append(out, merchantResponse(m, false))
	}
	return c.JSON(fiber.Map{"merchants": out})
}

func (h *AdminHandler) GetMerchant(c *fiber.Ctx) error {
	m, err := h.merchants.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get merchant"})
	}
	return c.JSON(merchantResponse(m, false))
}

func (h *AdminHandler) CreateMerchant(c *fiber.Ctx) error {
	var body struct {
		Name       string `json:"name"`
		Domain     string `json:"domain"`
		WebhookURL string `json:"webhook_url"`
		ReturnURL  string `json:"return_url"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	body.Name = strings.TrimSpace(body.Name)
	body.Domain = strings.TrimSpace(body.Domain)
	if body.Name == "" || body.Domain == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "name and domain required"})
	}

	slug := merchantSlug(body.Domain)
	m := &models.Merchant{
		ID:        security.NewID(),
		Name:      body.Name,
		Domain:    body.Domain,
		APIKey:    security.NewAPIKey("mk_" + slug),
		APISecret: "sk_" + security.NewAPISecret(),
		WebhookURL: body.WebhookURL,
		ReturnURL:  body.ReturnURL,
		Status:    models.MerchantStatusActive,
	}
	if err := h.merchants.Create(c.Context(), m); err != nil {
		if errors.Is(err, repository.ErrDuplicateOrder) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "merchant already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(merchantResponse(m, true))
}

func (h *AdminHandler) UpdateMerchant(c *fiber.Ctx) error {
	var body struct {
		Name       string `json:"name"`
		Domain     string `json:"domain"`
		WebhookURL string `json:"webhook_url"`
		ReturnURL  string `json:"return_url"`
		Status     string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if err := h.merchants.Update(c.Context(), c.Params("id"), repository.MerchantInput{
		Name: body.Name, Domain: body.Domain,
		WebhookURL: body.WebhookURL, ReturnURL: body.ReturnURL, Status: body.Status,
	}); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	m, _ := h.merchants.GetByID(c.Context(), c.Params("id"))
	return c.JSON(merchantResponse(m, false))
}

func (h *AdminHandler) RegenerateMerchantSecret(c *fiber.Ctx) error {
	secret, err := h.merchants.RegenerateAPISecret(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	m, _ := h.merchants.GetByID(c.Context(), c.Params("id"))
	return c.JSON(fiber.Map{
		"api_key":    m.APIKey,
		"api_secret": secret,
		"message":    "New secret generated. Update aMember plugin settings immediately.",
	})
}

func (h *AdminHandler) OnboardWebsite(c *fiber.Ctx) error {
	var body struct {
		Merchant struct {
			Name       string `json:"name"`
			Domain     string `json:"domain"`
			WebhookURL string `json:"webhook_url"`
			ReturnURL  string `json:"return_url"`
		} `json:"merchant"`
		Payment struct {
			Mode      string               `json:"mode"`
			ProfileID string               `json:"profile_id"`
			Profile   services.ProfileInput `json:"profile"`
		} `json:"payment"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	body.Merchant.Name = strings.TrimSpace(body.Merchant.Name)
	body.Merchant.Domain = strings.TrimSpace(body.Merchant.Domain)
	if body.Merchant.Name == "" || body.Merchant.Domain == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "merchant name and domain required"})
	}
	if body.Merchant.WebhookURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "webhook_url required"})
	}

	var profileID string
	mode := strings.ToLower(strings.TrimSpace(body.Payment.Mode))
	if mode == "" {
		mode = "existing"
	}

	switch mode {
	case "existing":
		if body.Payment.ProfileID == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "profile_id required for existing profile"})
		}
		if _, err := h.profiles.Get(c.Context(), body.Payment.ProfileID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "payment profile not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		profileID = body.Payment.ProfileID
	case "new":
		in := body.Payment.Profile
		if in.Name == "" || in.UPIID == "" || in.IMAPUser == "" || in.IMAPPassword == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "new profile requires name, upi_id, imap_user, imap_password",
			})
		}
		p, err := h.profiles.Create(c.Context(), in)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
		profileID = p.ID
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "payment.mode must be existing or new"})
	}

	slug := merchantSlug(body.Merchant.Domain)
	m := &models.Merchant{
		ID:               security.NewID(),
		Name:             body.Merchant.Name,
		Domain:           body.Merchant.Domain,
		APIKey:           security.NewAPIKey("mk_" + slug),
		APISecret:        "sk_" + security.NewAPISecret(),
		WebhookURL:       body.Merchant.WebhookURL,
		ReturnURL:        body.Merchant.ReturnURL,
		Status:           models.MerchantStatusActive,
		PaymentProfileID: profileID,
	}
	if err := h.merchants.Create(c.Context(), m); err != nil {
		if errors.Is(err, repository.ErrDuplicateOrder) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "merchant with this domain may already exist"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	profile, _ := h.profiles.Get(c.Context(), profileID)
	resp := fiber.Map{
		"merchant": merchantResponse(m, true),
		"checklist": fiber.Map{
			"amember_plugin_path": "app/application/default/plugins/payment/upipays.php",
			"hub_url":             "https://upays.in",
			"webhook_url":         m.WebhookURL,
		},
	}
	if profile != nil {
		resp["payment_profile"] = services.ProfileResponse(profile)
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

func merchantSlug(domain string) string {
	slug := strings.ReplaceAll(strings.Split(domain, ".")[0], "-", "")
	if slug == "" {
		return "merchant"
	}
	return slug
}

func (h *AdminHandler) AssignMerchantProfile(c *fiber.Ctx) error {
	var body struct {
		PaymentProfileID string `json:"payment_profile_id"`
	}
	if err := c.BodyParser(&body); err != nil || body.PaymentProfileID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "payment_profile_id required"})
	}
	if err := h.profiles.AssignMerchant(c.Context(), c.Params("id"), body.PaymentProfileID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AdminHandler) ListWebhooks(c *fiber.Ctx) error {
	limit, _ := strconv.Atoi(c.Query("limit", "25"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	items, total, err := h.webhooks.List(c.Context(), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list webhooks"})
	}
	out := make([]fiber.Map, 0, len(items))
	for _, w := range items {
		row := fiber.Map{
			"id": w.ID, "order_id": w.OrderID, "hub_order_id": w.HubOrderID,
			"merchant_name": w.MerchantName, "direction": w.Direction,
			"status": w.Status, "created_at": w.CreatedAt,
		}
		if w.ResponseCode != nil {
			row["response_code"] = *w.ResponseCode
		}
		out = append(out, row)
	}
	return c.JSON(fiber.Map{"webhooks": out, "total": total})
}

func (h *AdminHandler) ParserTypes(c *fiber.Ctx) error {
	types := make([]fiber.Map, 0, len(parser.AllParserTypes()))
	for _, p := range parser.AllParserTypes() {
		types = append(types, fiber.Map{
			"id":            p.ID,
			"label":         p.Label,
			"sender_filter": p.SenderFilter,
			"bank_code":     p.BankCode,
		})
	}
	return c.JSON(fiber.Map{"parser_types": types})
}

func merchantResponse(m *models.Merchant, revealSecret bool) fiber.Map {
	resp := fiber.Map{
		"id":                 m.ID,
		"name":               m.Name,
		"domain":             m.Domain,
		"api_key":            m.APIKey,
		"status":             m.Status,
		"webhook_url":        m.WebhookURL,
		"return_url":         m.ReturnURL,
		"payment_profile_id": m.PaymentProfileID,
		"created_at":         m.CreatedAt,
		"updated_at":         m.UpdatedAt,
	}
	if revealSecret {
		resp["api_secret"] = m.APISecret
	} else {
		resp["api_secret"] = security.MaskSecret(m.APISecret)
	}
	return resp
}

func (h *AdminHandler) ListSubscriptionPlans(c *fiber.Ctx) error {
	plans, err := h.subs.ListAllPlans(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list plans"})
	}
	out := make([]fiber.Map, 0, len(plans))
	for _, p := range plans {
		out = append(out, planAdminJSON(p))
	}
	return c.JSON(fiber.Map{"plans": out})
}

func planAdminJSON(p models.SubscriptionPlan) fiber.Map {
	return fiber.Map{
		"id": p.ID, "slug": p.Slug, "name": p.Name, "price_inr": p.PriceINR,
		"validity_days": p.ValidityDays, "order_limit": p.OrderLimit,
		"is_recommended": p.IsRecommended, "sort_order": p.SortOrder,
		"is_active": p.IsActive, "features_json": p.FeaturesJSON,
	}
}

func (h *AdminHandler) CreateSubscriptionPlan(c *fiber.Ctx) error {
	var body struct {
		Slug           string  `json:"slug"`
		Name           string  `json:"name"`
		PriceINR       float64 `json:"price_inr"`
		ValidityDays   int     `json:"validity_days"`
		OrderLimit     int     `json:"order_limit"`
		IsRecommended  bool    `json:"is_recommended"`
		SortOrder      int     `json:"sort_order"`
		IsActive       bool    `json:"is_active"`
		FeaturesJSON   string  `json:"features_json"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Slug == "" || body.Name == "" || body.OrderLimit <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "slug, name, order_limit required"})
	}
	if body.ValidityDays <= 0 {
		body.ValidityDays = 28
	}
	p, err := h.subs.CreatePlan(c.Context(), repository.PlanInput{
		Slug: body.Slug, Name: body.Name, PriceINR: body.PriceINR,
		ValidityDays: body.ValidityDays, OrderLimit: body.OrderLimit,
		IsRecommended: body.IsRecommended, SortOrder: body.SortOrder,
		IsActive: body.IsActive, FeaturesJSON: body.FeaturesJSON,
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateOrder) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "slug already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(planAdminJSON(*p))
}

func (h *AdminHandler) UpdateSubscriptionPlan(c *fiber.Ctx) error {
	var body struct {
		Slug           string  `json:"slug"`
		Name           string  `json:"name"`
		PriceINR       float64 `json:"price_inr"`
		ValidityDays   int     `json:"validity_days"`
		OrderLimit     int     `json:"order_limit"`
		IsRecommended  bool    `json:"is_recommended"`
		SortOrder      int     `json:"sort_order"`
		IsActive       bool    `json:"is_active"`
		FeaturesJSON   string  `json:"features_json"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	p, err := h.subs.UpdatePlan(c.Context(), c.Params("id"), repository.PlanInput{
		Slug: body.Slug, Name: body.Name, PriceINR: body.PriceINR,
		ValidityDays: body.ValidityDays, OrderLimit: body.OrderLimit,
		IsRecommended: body.IsRecommended, SortOrder: body.SortOrder,
		IsActive: body.IsActive, FeaturesJSON: body.FeaturesJSON,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		if errors.Is(err, repository.ErrDuplicateOrder) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "slug already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(planAdminJSON(*p))
}

func (h *AdminHandler) GetMerchantSubscription(c *fiber.Ctx) error {
	usage, err := h.subs.GetUsage(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, services.ErrNoSubscription) {
			return c.JSON(fiber.Map{"subscription": nil})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load subscription"})
	}
	return c.JSON(fiber.Map{"subscription": usage})
}

func (h *AdminHandler) ActivateMerchantSubscription(c *fiber.Ctx) error {
	var body struct {
		PlanID string `json:"plan_id"`
		Notes  string `json:"notes"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.PlanID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "plan_id required"})
	}
	adminEmail, _ := c.Locals("admin_email").(string)
	if err := h.subs.ActivatePlan(c.Context(), c.Params("id"), body.PlanID, adminEmail, body.Notes); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "plan or merchant not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	usage, _ := h.subs.GetUsage(c.Context(), c.Params("id"))
	return c.JSON(fiber.Map{"ok": true, "subscription": usage})
}

func cmsPageAdminJSON(p *models.CMSPage) fiber.Map {
	return fiber.Map{
		"id": p.ID, "slug": p.Slug, "title": p.Title, "meta_description": p.MetaDescription,
		"body_html": p.BodyHTML, "status": p.Status, "show_in_nav": p.ShowInNav,
		"nav_label": p.NavLabel, "sort_order": p.SortOrder,
		"created_at": p.CreatedAt, "updated_at": p.UpdatedAt,
	}
}

func (h *AdminHandler) ListCMSPages(c *fiber.Ctx) error {
	pages, err := h.cms.List(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list pages"})
	}
	out := make([]fiber.Map, 0, len(pages))
	for i := range pages {
		out = append(out, cmsPageAdminJSON(&pages[i]))
	}
	return c.JSON(fiber.Map{"pages": out})
}

func (h *AdminHandler) GetCMSPage(c *fiber.Ctx) error {
	p, err := h.cms.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cmsPageAdminJSON(p))
}

func (h *AdminHandler) CreateCMSPage(c *fiber.Ctx) error {
	var body struct {
		Slug            string `json:"slug"`
		Title           string `json:"title"`
		MetaDescription string `json:"meta_description"`
		BodyHTML        string `json:"body_html"`
		Status          string `json:"status"`
		ShowInNav       bool   `json:"show_in_nav"`
		NavLabel        string `json:"nav_label"`
		SortOrder       int    `json:"sort_order"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	if body.Slug == "" || body.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "slug and title required"})
	}
	p, err := h.cms.Create(c.Context(), repository.CMSPageInput{
		Slug: body.Slug, Title: body.Title, MetaDescription: body.MetaDescription,
		BodyHTML: body.BodyHTML, Status: body.Status, ShowInNav: body.ShowInNav,
		NavLabel: body.NavLabel, SortOrder: body.SortOrder,
	})
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateOrder) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "slug already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(cmsPageAdminJSON(p))
}

func (h *AdminHandler) UpdateCMSPage(c *fiber.Ctx) error {
	var body struct {
		Slug            string `json:"slug"`
		Title           string `json:"title"`
		MetaDescription string `json:"meta_description"`
		BodyHTML        string `json:"body_html"`
		Status          string `json:"status"`
		ShowInNav       bool   `json:"show_in_nav"`
		NavLabel        string `json:"nav_label"`
		SortOrder       int    `json:"sort_order"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}
	p, err := h.cms.Update(c.Context(), c.Params("id"), repository.CMSPageInput{
		Slug: body.Slug, Title: body.Title, MetaDescription: body.MetaDescription,
		BodyHTML: body.BodyHTML, Status: body.Status, ShowInNav: body.ShowInNav,
		NavLabel: body.NavLabel, SortOrder: body.SortOrder,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		if errors.Is(err, repository.ErrDuplicateOrder) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "slug already exists"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(cmsPageAdminJSON(p))
}

func (h *AdminHandler) DeleteCMSPage(c *fiber.Ctx) error {
	if err := h.cms.Delete(c.Context(), c.Params("id")); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *AdminHandler) PreviewCMSPage(c *fiber.Ctx) error {
	p, err := h.cms.GetByID(c.Context(), c.Params("id"))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.SendString(RenderCMSPreview(p))
}
