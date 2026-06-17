package api

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

func registerDocsSite(app *fiber.App) {
	root := "./web/docs"
	if _, err := os.Stat(root + "/index.html"); os.IsNotExist(err) {
		return
	}

	app.Static("/docs/assets", root+"/assets")

	pages := map[string]string{
		"/docs":                "/index.html",
		"/docs/":               "/index.html",
		"/docs/auth":           "/auth.html",
		"/docs/create-order":   "/create-order.html",
		"/docs/verify-order":   "/verify-order.html",
		"/docs/webhooks":       "/webhooks.html",
		"/docs/checkout":       "/checkout.html",
		"/docs/errors":         "/errors.html",
		"/docs/sdks/php":       "/sdks/php.html",
		"/docs/sdks/node":      "/sdks/node.html",
		"/docs/sdks/python":    "/sdks/python.html",
		"/docs/plugins/amember": "/plugins/amember.html",
	}
	for route, file := range pages {
		route, file := route, file
		app.Get(route, func(c *fiber.Ctx) error {
			return c.SendFile(root + file)
		})
	}

	app.Get("/docs/upipays.postman_collection.json", func(c *fiber.Ctx) error {
		return c.SendFile(root + "/upipays.postman_collection.json")
	})
}
