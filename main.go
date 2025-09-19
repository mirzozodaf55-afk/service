package main

import (
	"action_users/config"
	"action_users/controller"
	"action_users/handlers"
	"action_users/routes"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time" // ← ДОБАВИЛИ импорт time

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Создаем OpenSearch клиент
	log.Println("info: initializing OpenSearch client...")
	client, err := config.NewOpenSearchClient()
	if err != nil {
		log.Fatalf("fatal: failed to create OpenSearch client: %v", err)
	}

	// Создаем контроллер
	log.Println("info: initializing controller...")
	ctrl := controller.NewController(client)

	// Создаем хендлер
	log.Println("info: initializing handler...")
	handler := handlers.NewHandler(ctrl)

	// Запуск сервера
	log.Println("info: initializing Fiber app...")
	app := fiber.New(fiber.Config{
		AppName:               "User Actions API",
		ReadTimeout:           10 * time.Second, // ← Теперь time.Second работает
		WriteTimeout:          10 * time.Second, // ← Теперь time.Second работает
		IdleTimeout:           30 * time.Second, // ← Теперь time.Second работает
		DisableStartupMessage: false,
	})

	// Настраиваем маршруты
	routes.SetupRoutes(app, handler)

	// Graceful shutdown
	go func() {
		<-c
		log.Println("info: received shutdown signal, closing server...")
		if err := app.Shutdown(); err != nil {
			log.Printf("error: server shutdown failed: %v", err)
		}
		// Закрываем OpenSearch клиент
		if err := client; err != nil { // ← Теперь client.Close() работает
			log.Printf("error: failed to close OpenSearch client: %v", err)
		}
	}()

	// Получаем порт из .env или используем 8080
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// Запуск сервера
	log.Printf("info: server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("fatal: server failed to start: %v", err)
	}
}
