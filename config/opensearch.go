package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/opensearch-project/opensearch-go"
)

type OpenSearchConfig struct {
	Host     string
	Username string
	Password string
}

func LoadOpenSearchConfig() (*OpenSearchConfig, error) {
	if err := godotenv.Load(); err != nil {
		log.Printf("warning: .env file not found, using environment variables")
	}
	host := os.Getenv("OPENSEARCH_HOST")
	username := os.Getenv("OPENSEARCH_USERNAME")
	password := os.Getenv("OPENSEARCH_PASSWORD")

	if host == "" {
		return nil, fmt.Errorf("OPENSEARCH_HOST is required")
	}
	if username == "" {
		return nil, fmt.Errorf("OPENSEARCH_USERNAME is required")
	}
	if password == "" {
		return nil, fmt.Errorf("OPENSEARCH_PASSWORD is required")
	}

	config := &OpenSearchConfig{
		Host:     host,
		Username: username,
		Password: password,
	}

	log.Printf("info: OpenSearch config loaded - host: %s, username: %s", host, username)
	return config, nil
}

func NewOpenSearchClient() (*opensearch.Client, error) {
	config, err := LoadOpenSearchConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenSearch config: %w", err)
	}

	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{config.Host},
		Username:  config.Username,
		Password:  config.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenSearch client: %w", err)
	}

	_, err = client.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping OpenSearch: %w", err)
	}

	log.Println("info: OpenSearch client connected successfully")
	return client, nil
}
