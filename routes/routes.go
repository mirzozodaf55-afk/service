package routes

import (
	"action_users/handlers"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes настраивает маршруты для Fiber.
func SetupRoutes(app *fiber.App, handler *handlers.Handler) {
	// Health check
	app.Get("/health", handler.HealthCheck)

	// Основной эндпоинт
	app.Get("/process-users", handler.ProcessUsers)

	// Корневой маршрут
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "User Actions API",
			"endpoints": fiber.Map{
				"health":  "/health",
				"process": "/process-users?months=1&countryId=213&page=5&limit=50",
			},
		})
	})
}
