package middleware

import (
	"strings"

	"github.com/sagartiwari-net/upays.in/payment-hub/internal/security"
	"github.com/gofiber/fiber/v2"
)

func AdminAuth(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization"})
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == auth {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization format"})
		}
		claims, err := security.ParseAdminToken(token, jwtSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		c.Locals("admin_id", claims.AdminID)
		c.Locals("admin_email", claims.Email)
		c.Locals("admin_role", claims.Role)
		return c.Next()
	}
}
