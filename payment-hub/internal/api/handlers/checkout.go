package handlers

import (
	"errors"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/payment"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/repository"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/services"
	"github.com/gofiber/fiber/v2"
)

type CheckoutHandler struct {
	provider string
	upi      *services.UPIService
	phonepe  *services.PaymentService
}

func NewCheckoutHandler(provider string, upi *services.UPIService, phonepe *services.PaymentService) *CheckoutHandler {
	return &CheckoutHandler{provider: provider, upi: upi, phonepe: phonepe}
}

func (h *CheckoutHandler) Show(c *fiber.Ctx) error {
	token := c.Params("token")

	if h.provider == payment.ProviderUPIEmail {
		view, _, err := h.upi.GetCheckout(c.Context(), token)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return c.Status(fiber.StatusNotFound).SendString("Payment link not found")
			}
			return c.Status(fiber.StatusInternalServerError).SendString("Something went wrong")
		}
		html, err := services.RenderUPICheckoutHTML(view)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Render error")
		}
		c.Set("Content-Type", "text/html; charset=utf-8")
		c.Set("X-Frame-Options", "DENY")
		return c.SendString(html)
	}

	view, _, err := h.phonepe.GetCheckout(c.Context(), token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).SendString("Payment link not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Something went wrong")
	}

	html, err := services.RenderCheckoutHTML(view)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Render error")
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	c.Set("X-Frame-Options", "DENY")
	return c.SendString(html)
}

func (h *CheckoutHandler) QR(c *fiber.Ctx) error {
	if h.upi == nil {
		return c.Status(fiber.StatusNotFound).SendString("not found")
	}
	token := c.Params("token")
	png, err := h.upi.GetQRImage(c.Context(), token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).SendString("not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("qr error")
	}
	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "no-store")
	return c.Send(png)
}

func (h *CheckoutHandler) Status(c *fiber.Ctx) error {
	if h.upi == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not available"})
	}
	token := c.Params("token")
	resp, err := h.upi.GetStatus(c.Context(), token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(resp)
}

func (h *CheckoutHandler) SubmitUTR(c *fiber.Ctx) error {
	if h.upi == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not available"})
	}
	token := c.Params("token")
	var body struct {
		UTR string `json:"utr"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid body"})
	}

	err := h.upi.SubmitUTR(c.Context(), token, body.UTR)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		case errors.Is(err, services.ErrInvalidInput):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid utr"})
		case errors.Is(err, services.ErrOrderExpired):
			return c.Status(fiber.StatusGone).JSON(fiber.Map{"error": "payment link expired"})
		case errors.Is(err, services.ErrOrderNotPayable):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "order not payable"})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (h *CheckoutHandler) Pay(c *fiber.Ctx) error {
	token := c.Params("token")
	payURL, err := h.phonepe.InitiatePayment(c.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			return c.Status(fiber.StatusNotFound).SendString("Payment link not found")
		case errors.Is(err, services.ErrOrderExpired):
			return c.Status(fiber.StatusGone).SendString("Payment link expired")
		case errors.Is(err, services.ErrOrderNotPayable):
			return c.Redirect(h.phonepe.CheckoutURL(token), fiber.StatusTemporaryRedirect)
		default:
			return c.Status(fiber.StatusBadGateway).SendString("Unable to start PhonePe payment: " + err.Error())
		}
	}
	return c.Redirect(payURL, fiber.StatusTemporaryRedirect)
}

func (h *CheckoutHandler) Return(c *fiber.Ctx) error {
	token := c.Params("token")
	base64Resp := c.FormValue("response")
	if base64Resp == "" {
		base64Resp = c.Query("response")
	}
	xVerify := c.Get("X-VERIFY")

	redirectURL, err := h.phonepe.HandleReturn(c.Context(), token, base64Resp, xVerify)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return c.Status(fiber.StatusNotFound).SendString("Payment not found")
		}
		return c.Status(fiber.StatusInternalServerError).SendString("Unable to process return")
	}
	return c.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}

type PhonePeWebhookHandler struct {
	payments *services.PaymentService
}

func NewPhonePeWebhookHandler(payments *services.PaymentService) *PhonePeWebhookHandler {
	return &PhonePeWebhookHandler{payments: payments}
}

func (h *PhonePeWebhookHandler) Handle(c *fiber.Ctx) error {
	var body struct {
		Response string `json:"response"`
	}
	if err := c.BodyParser(&body); err != nil || body.Response == "" {
		body.Response = string(c.Body())
	}

	xVerify := c.Get("X-VERIFY")
	if err := h.payments.HandlePhonePeWebhook(c.Context(), body.Response, xVerify); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}
	return c.JSON(fiber.Map{"success": true})
}
