package api

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

func registerPublicSite(app *fiber.App) {
	root := "./web/public"
	if _, err := os.Stat(root + "/index.html"); os.IsNotExist(err) {
		return
	}

	app.Static("/assets", root+"/assets")

	pages := map[string]string{
		"/":         "/index.html",
		"/pricing":  "/pricing/index.html",
		"/faq":      "/faq/index.html",
		"/contact":  "/contact/index.html",
		"/terms":    "/terms/index.html",
		"/privacy":  "/privacy/index.html",
	}
	for route, file := range pages {
		route, file := route, file
		app.Get(route, func(c *fiber.Ctx) error {
			return c.SendFile(root + file)
		})
	}

	app.Get("/register", func(c *fiber.Ctx) error {
		return c.Redirect("/dashboard/register", fiber.StatusTemporaryRedirect)
	})
}
