package routes

import (
	"action_users/handlers"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, handler *handlers.Handler) {
	app.Get("/health", handler.HealthCheck)

	app.Get("/process-users", handler.ProcessUsers)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "User Actions API",
			"endpoints": fiber.Map{
				"health": "/health",
				"process-users": fiber.Map{
					"method": "GET",
					"path":   "/process-users",
					"parameters": fiber.Map{
						"months":    "Количество месяцев для определения неактивности И вычисления порога реактивации (default: 1)",
						"countryId": "ID страны (default: 0 - все страны)",
						"page":      "Номер страницы (default: 1)",
						"limit":     "Количество записей на странице (default: 100, max: 1000)",
					},
					"example": "/process-users?months=3&countryId=213&page=1&limit=50",
					"logic":   "Для каждого неактивного пользователя: lastActivity - months = reactivationThreshold",
				},
			},
		})
	})
}
