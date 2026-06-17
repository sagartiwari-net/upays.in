package api

import (
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

func registerAdminUI(app *fiber.App) {
	diskRoot := "./web/admin"
	if _, err := os.Stat(diskRoot + "/index.html"); os.IsNotExist(err) {
		return
	}

	app.Use("/admin", filesystem.New(filesystem.Config{
		Root:  http.Dir(diskRoot),
		Index: "index.html",
		Next:  skipAdminAPI,
	}))
	app.Get("/admin/*", adminSPAFallback(diskRoot))
}

func skipAdminAPI(c *fiber.Ctx) bool {
	return strings.HasPrefix(c.Path(), "/admin/api")
}

func adminSPAFallback(diskRoot string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if skipAdminAPI(c) {
			return c.Next()
		}
		p := c.Path()
		if strings.Contains(strings.TrimPrefix(p, "/admin/"), ".") {
			return c.Next()
		}
		return c.SendFile(diskRoot + "/index.html")
	}
}
