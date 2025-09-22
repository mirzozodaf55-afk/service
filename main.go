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
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	log.Println("info: initializing OpenSearch client...")
	client, err := config.NewOpenSearchClient()
	if err != nil {
		log.Fatalf("fatal: failed to create OpenSearch client: %v", err)
	}

	ctrl := controller.NewController(client)

	handler := handlers.NewHandler(ctrl)

	app := fiber.New(fiber.Config{
		AppName:               "User Actions API",
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		DisableStartupMessage: false,
		IdleTimeout:           30 * time.Second,
	})

	routes.SetupRoutes(app, handler)

	go func() {
		<-c
		log.Println("info: received shutdown signal, closing server...")
		if err := app.Shutdown(); err != nil {
			log.Printf("error: server shutdown failed: %v", err)
		}
		if err := client; err != nil {
			log.Printf("error: failed to close OpenSearch client: %v", err)
		}
	}()

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("info: server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("fatal: server failed to start: %v", err)
	}
}
