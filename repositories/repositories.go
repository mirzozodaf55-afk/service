package repositories

import (
	"action_users/constants"
	"action_users/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	_ "sync"

	"github.com/opensearch-project/opensearch-go"
)

// toInt64 конвертирует строку в int64.
func toInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// doSearch выполняет поисковый запрос и возвращает SearchResponse.
func doSearch(client *opensearch.Client, index string, query map[string]interface{}) (*models.SearchResponse, error) {
	body, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	res, err := client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex(index),
		client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var sr models.SearchResponse
	if err := json.NewDecoder(res.Body).Decode(&sr); err != nil {
		return nil, err
	}
	return &sr, nil
}

// GetUserIds получает список userId из clients-searcher.
func GetUserIds(client *opensearch.Client, from, size, countryId int) ([]string, error) {
	query := map[string]interface{}{
		"_source": []string{"stats.userId"},
		"from":    from,
		"size":    size,
	}

	if countryId != 0 {
		query["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]interface{}{"user.countryId": countryId}},
				},
			},
		}
	} else {
		query["query"] = map[string]interface{}{"match_all": map[string]interface{}{}}
	}

	sr, err := doSearch(client, "clients-searcher", query)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, hit := range sr.Hits.Hits {
		if stats, ok := hit.Source["stats"].(map[string]interface{}); ok {
			if idf, ok := stats["userId"].(float64); ok {
				ids = append(ids, fmt.Sprintf("%.0f", idf))
			}
		}
	}
	return ids, nil
}

// GetClientById получает данные клиента по userId.
func GetClientById(client *opensearch.Client, userIdStr string, countryId int) (map[string]interface{}, error) {
	userIdInt, err := toInt64(userIdStr)
	if err != nil {
		return nil, err
	}
	must := []map[string]interface{}{
		{"term": map[string]interface{}{"stats.userId": userIdInt}},
	}
	if countryId != 0 {
		must = append(must, map[string]interface{}{"term": map[string]interface{}{"user.countryId": countryId}})
	}

	query := map[string]interface{}{
		"size": 1,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
	}

	sr, err := doSearch(client, "clients-searcher", query)
	if err != nil {
		log.Printf("error: getClientById(%s) search error: %v", userIdStr, err)
		return nil, err
	}

	if len(sr.Hits.Hits) > 0 {
		source := sr.Hits.Hits[0].Source
		if user, ok := source["user"].(map[string]interface{}); !ok || user["createdAt"] == nil {
			log.Printf("warn: user document missing createdAt for userId: %s", userIdStr)
		}
		return source, nil
	}

	if countryId != 0 {
		log.Printf("debug: client not found with country filter for %s, trying without country", userIdStr)
		query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must"] = []map[string]interface{}{
			{"term": map[string]interface{}{"stats.userId": userIdInt}},
		}
		sr, err = doSearch(client, "clients-searcher", query)
		if err != nil {
			return nil, err
		}
		if len(sr.Hits.Hits) > 0 {
			log.Printf("debug: found client without country filter for %s", userIdStr)
			return sr.Hits.Hits[0].Source, nil
		}
	}

	return nil, nil
}

// GetActionsFromIndexNoCountry получает действия без фильтра по стране.
func GetActionsFromIndexNoCountry(client *opensearch.Client, userIdStr string, index string, size int) ([]map[string]interface{}, error) {
	userIdInt, err := toInt64(userIdStr)
	if err != nil {
		return nil, err
	}

	query := map[string]interface{}{
		"size": size,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{"term": map[string]interface{}{"user.id": userIdInt}},
				},
			},
		},
		"sort": []map[string]interface{}{
			{constants.Indices[index] + ".createdAt": map[string]string{"order": "desc"}},
		},
	}

	sr, err := doSearch(client, index, query)
	if err != nil {
		return nil, err
	}
	var out []map[string]interface{}
	for _, h := range sr.Hits.Hits {
		out = append(out, h.Source)
	}
	return out, nil
}

// GetActionsFromIndex получает последние N документов по индексу для userId.
func GetActionsFromIndex(client *opensearch.Client, userIdStr string, index string, size, countryId int) ([]map[string]interface{}, error) {
	userIdInt, err := toInt64(userIdStr)
	if err != nil {
		return nil, err
	}

	must := []map[string]interface{}{
		{"term": map[string]interface{}{"user.id": userIdInt}},
	}
	if countryId != 0 {
		must = append(must, map[string]interface{}{"term": map[string]interface{}{"user.countryId": countryId}})
	}

	sortField := constants.Indices[index] + ".createdAt"
	query := map[string]interface{}{
		"size": size,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": must,
			},
		},
		"sort": []map[string]interface{}{
			{sortField: map[string]string{"order": "desc"}},
		},
	}

	sr, err := doSearch(client, index, query)
	if err != nil {
		return nil, err
	}
	var out []map[string]interface{}
	for _, h := range sr.Hits.Hits {
		out = append(out, h.Source)
	}
	return out, nil
}

// GetCreatedAt извлекает createdAt из документа.
func GetCreatedAt(source map[string]interface{}) int64 {
	if source == nil {
		return 0
	}
	for _, key := range []string{"entity", "withdrawal", "bet", "card"} {
		if obj, ok := source[key].(map[string]interface{}); ok {
			if ts, ok := obj["createdAt"].(float64); ok {
				return int64(ts)
			}
		}
	}
	if ts, ok := source["createdAt"].(float64); ok {
		return int64(ts)
	}
	return 0
}
