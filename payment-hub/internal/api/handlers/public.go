package handlers

import (
	"html"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/models"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
)

type PublicHandler struct {
	plans *repository.SubscriptionRepository
	pages *repository.CMSPageRepository
}

func NewPublicHandler(plans *repository.SubscriptionRepository, pages *repository.CMSPageRepository) *PublicHandler {
	return &PublicHandler{plans: plans, pages: pages}
}

func (h *PublicHandler) ListPlans(c *fiber.Ctx) error {
	plans, err := h.plans.ListPlans(c.Context(), true)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load plans"})
	}
	out := make([]fiber.Map, 0, len(plans))
	for _, p := range plans {
		if p.Slug == "trial" {
			continue
		}
		out = append(out, planPublicJSON(p))
	}
	return c.JSON(fiber.Map{"plans": out})
}

func (h *PublicHandler) ListNavPages(c *fiber.Ctx) error {
	pages, err := h.pages.ListPublishedNav(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load pages"})
	}
	out := make([]fiber.Map, 0, len(pages))
	for _, p := range pages {
		label := p.NavLabel
		if label == "" {
			label = p.Title
		}
		out = append(out, fiber.Map{
			"slug":  p.Slug,
			"label": label,
			"url":   "/" + p.Slug,
		})
	}
	return c.JSON(fiber.Map{"pages": out})
}

func (h *PublicHandler) GetPage(c *fiber.Ctx) error {
	page, err := h.pages.GetPublishedBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		if err == repository.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "page not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to load page"})
	}
	return c.JSON(fiber.Map{"page": cmsPagePublicJSON(page)})
}

func (h *PublicHandler) ServeCMSPage(c *fiber.Ctx) error {
	slug := strings.ToLower(strings.TrimSpace(c.Params("slug")))
	if slug == "" || isReservedPublicSlug(slug) {
		return c.SendStatus(fiber.StatusNotFound)
	}
	page, err := h.pages.GetPublishedBySlug(c.Context(), slug)
	if err != nil {
		return c.SendStatus(fiber.StatusNotFound)
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.SendString(renderCMSHTML(page))
}

func planPublicJSON(p models.SubscriptionPlan) fiber.Map {
	return fiber.Map{
		"id": p.ID, "slug": p.Slug, "name": p.Name, "price_inr": p.PriceINR,
		"validity_days": p.ValidityDays, "order_limit": p.OrderLimit,
		"is_recommended": p.IsRecommended, "features_json": p.FeaturesJSON,
	}
}

func cmsPagePublicJSON(p *models.CMSPage) fiber.Map {
	return fiber.Map{
		"slug": p.Slug, "title": p.Title, "meta_description": p.MetaDescription,
		"body_html": p.BodyHTML, "show_in_nav": p.ShowInNav, "nav_label": p.NavLabel,
	}
}

func RenderCMSPreview(page *models.CMSPage) string {
	return renderCMSHTML(page)
}

func renderCMSHTML(page *models.CMSPage) string {
	title := html.EscapeString(page.Title)
	meta := html.EscapeString(page.MetaDescription)
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>` + title + ` — UPIPays</title>
  <meta name="description" content="` + meta + `">
  <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
  <link rel="stylesheet" href="/assets/css/site.css">
</head>
<body>
  <nav class="site-nav">
    <div class="container nav-inner">
      <a href="/" class="logo"><span class="logo-icon"><i class="fas fa-bolt"></i></span> UPIPays</a>
      <div class="nav-links" id="site-nav-links">
        <a href="/">Home</a>
        <a href="/pricing">Pricing</a>
        <a href="/faq">FAQ</a>
        <a href="/contact">Contact</a>
        <a href="/admin/login" class="btn btn-outline">Login</a>
        <a href="/dashboard/register" class="btn btn-primary">Get Started</a>
      </div>
    </div>
  </nav>
  <section style="padding: 64px 0 48px;">
    <div class="container cms-content">
      <h1 class="section-title">` + title + `</h1>
      ` + page.BodyHTML + `
    </div>
  </section>
  <footer class="site-footer">
    <div class="container"><div class="footer-bottom">&copy; 2026 UPIPays — upays.in</div></div>
  </footer>
  <script src="/assets/js/nav.js"></script>
</body>
</html>`
}

func isReservedPublicSlug(slug string) bool {
	reserved := []string{
		"admin", "dashboard", "merchant", "api", "pay", "health", "register",
		"assets", "pricing", "faq", "contact", "terms", "privacy", "public", "docs",
	}
	for _, r := range reserved {
		if slug == r {
			return true
		}
	}
	return false
}
