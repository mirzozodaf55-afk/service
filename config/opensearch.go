package config

import (
	"github.com/opensearch-project/opensearch-go"
)

// NewOpenSearchClient создает новый клиент OpenSearch.
func NewOpenSearchClient() (*opensearch.Client, error) {
	return opensearch.NewClient(opensearch.Config{
		Addresses: []string{"https://vpc-formula55-yf46aaczgealfueyceomyztxgm.eu-west-1.es.amazonaws.com"},
		Username:  "admin",
		Password:  "Dev4F55DyuTJK!",
	})
}
