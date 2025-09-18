package main

import (
	"action_users/config"
	"action_users/controller"
	"action_users/handlers"
	"action_users/routes"
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Создаем OpenSearch клиент
	client, err := config.NewOpenSearchClient()
	if err != nil {
		log.Fatalf("Error creating client: %s", err)
	}

	// Создаем контроллер
	ctrl := controller.NewController(client)

	// Создаем хендлер
	handler := handlers.NewHandler(ctrl)

	// Запуск сервера
	app := fiber.New()

	// Настраиваем маршруты
	routes.SetupRoutes(app, handler)

	log.Println("Server starting on :8080")
	log.Fatal(app.Listen(":8080"))
}
