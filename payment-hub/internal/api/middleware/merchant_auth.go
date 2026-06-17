package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
)

func MerchantPortalJWT(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization"})
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization format"})
		}
		claims, err := security.ParseMerchantToken(token, jwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		c.Locals("merchant_user_id", claims.UserID)
		c.Locals("merchant_id", claims.MerchantID)
		c.Locals("merchant_email", claims.Email)
		return c.Next()
	}
}
