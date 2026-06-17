package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

func registerMerchantUI(app *fiber.App) {
	diskRoot := "./web/merchant"
	if _, err := os.Stat(diskRoot + "/index.html"); os.IsNotExist(err) {
		return
	}

	app.Use("/dashboard", filesystem.New(filesystem.Config{
		Root:  http.Dir(diskRoot),
		Index: "index.html",
		Next:  skipMerchantAPI,
	}))
	app.Get("/dashboard/*", merchantSPAFallback(diskRoot))
}

func skipMerchantAPI(c *fiber.Ctx) bool {
	return strings.HasPrefix(c.Path(), "/merchant/api")
}

func merchantSPAFallback(diskRoot string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if skipMerchantAPI(c) {
			return c.Next()
		}
		p := c.Path()
		if strings.Contains(strings.TrimPrefix(p, "/dashboard/"), ".") {
			return c.Next()
		}
		return c.SendFile(diskRoot + "/index.html")
	}
}
