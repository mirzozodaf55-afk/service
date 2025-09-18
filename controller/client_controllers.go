package controller

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"action_users/constants"
	"action_users/models"
	"action_users/repositories"

	"github.com/opensearch-project/opensearch-go"
)

// Controller управляет бизнес-логикой обработки данных клиентов.
type Controller struct {
	client *opensearch.Client // приватное поле
}

// NewController создает новый экземпляр Controller.
func NewController(client *opensearch.Client) *Controller {
	return &Controller{client: client}
}

// Client возвращает OpenSearch клиент (геттер)
func (c *Controller) Client() *opensearch.Client {
	return c.client
}

// GetLastTwoActionsForUser получает последние два действия пользователя.
func (c *Controller) GetLastTwoActionsForUser(userId string, countryId int) ([]map[string]interface{}, error) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	all := []map[string]interface{}{}

	for idx := range constants.Indices {
		wg.Add(1)
		go func(index string) {
			defer wg.Done()
			var acts []map[string]interface{}
			var err error

			acts, err = repositories.GetActionsFromIndex(c.client, userId, index, 2, countryId)
			if err != nil || len(acts) == 0 {
				acts, err = repositories.GetActionsFromIndexNoCountry(c.client, userId, index, 2)
			}

			if err != nil {
				log.Printf("warn: getActionsFromIndex %s err: %v", index, err)
				return
			}
			if len(acts) == 0 {
				return
			}
			mu.Lock()
			all = append(all, acts...)
			mu.Unlock()
		}(idx)
	}
	wg.Wait()

	if len(all) == 0 {
		return nil, nil
	}
	sort.Slice(all, func(i, j int) bool {
		return repositories.GetCreatedAt(all[i]) > repositories.GetCreatedAt(all[j])
	})
	if len(all) > 2 {
		all = all[:2]
	}
	return all, nil
}

// GetLastActionFromIndices получает последнее действие из списка индексов.
func (c *Controller) GetLastActionFromIndices(userId string, indicesList []string, countryId int) (map[string]interface{}, error) {
	var best map[string]interface{}
	var bestTs int64

	for _, idx := range indicesList {
		var srcs []map[string]interface{}
		var err error

		srcs, err = repositories.GetActionsFromIndex(c.client, userId, idx, 1, countryId)
		if err != nil || len(srcs) == 0 {
			srcs, err = repositories.GetActionsFromIndexNoCountry(c.client, userId, idx, 1)
		}

		if err != nil {
			log.Printf("warn: get last from index %s err: %v", idx, err)
			continue
		}
		if len(srcs) == 0 {
			continue
		}
		ts := repositories.GetCreatedAt(srcs[0])
		if ts > bestTs {
			bestTs = ts
			best = srcs[0]
		}
	}
	return best, nil
}

// CheckUserActionsInterval проверяет интервал между действиями.
func (c *Controller) CheckUserActionsInterval(actions []map[string]interface{}, frontInterval int) bool {
	if len(actions) == 0 {
		return false
	}
	first := repositories.GetCreatedAt(actions[0])
	if first == 0 {
		return false
	}

	var compareTime time.Time
	if len(actions) == 1 {
		compareTime = time.Now()
	} else {
		second := repositories.GetCreatedAt(actions[1])
		if second == 0 {
			compareTime = time.Now()
		} else {
			compareTime = time.Unix(second, 0)
		}
	}

	firstTime := time.Unix(first, 0)
	if firstTime.Before(compareTime) {
		firstTime, compareTime = compareTime, firstTime
	}
	diffMonths := int(firstTime.Sub(compareTime).Hours() / 24 / 30)
	return diffMonths >= frontInterval
}

// BuildClientData создает объект ClientData из данных и действий.
func (c *Controller) BuildClientData(clientData map[string]interface{}, topUpSrc, betSrc, withdrawalSrc map[string]interface{}, frontCountryId int, userId string) models.ClientData {
	cd := models.ClientData{
		Account:        models.Account{ActiveWallet: "", Balance: 0, CurrencyId: 0},
		LastTopUp:      repositories.GetCreatedAt(topUpSrc),
		LastBet:        repositories.GetCreatedAt(betSrc),
		LastWithdrawal: repositories.GetCreatedAt(withdrawalSrc),
		CreatedAt:      0,
		UserId:         userId,
		LastActivity:   0,
	}

	var maxActivity int64
	if cd.LastTopUp > maxActivity {
		maxActivity = cd.LastTopUp
	}
	if cd.LastBet > maxActivity {
		maxActivity = cd.LastBet
	}
	if cd.LastWithdrawal > maxActivity {
		maxActivity = cd.LastWithdrawal
	}

	if maxActivity > 0 {
		cd.LastActivity = maxActivity
		lastActionType := "UNKNOWN"
		if maxActivity == cd.LastTopUp && cd.LastTopUp > 0 {
			lastActionType = "TOP_UP"
		} else if maxActivity == cd.LastBet && cd.LastBet > 0 {
			lastActionType = "BET"
		} else if maxActivity == cd.LastWithdrawal && cd.LastWithdrawal > 0 {
			lastActionType = "WITHDRAWAL"
		}
		log.Printf("debug: user %s last activity: %s (%s) - topup:%d bet:%d withdrawal:%d",
			userId, time.Unix(maxActivity, 0).Format("2006-01-02"), lastActionType,
			cd.LastTopUp, cd.LastBet, cd.LastWithdrawal)
	}

	if clientData != nil {
		if user, ok := clientData["user"].(map[string]interface{}); ok {
			if createdAt, ok := user["createdAt"].(float64); ok && createdAt > 0 {
				cd.CreatedAt = int64(createdAt)
			} else if cd.CreatedAt == 0 && maxActivity > 0 {
				var earliestTs int64
				if cd.LastTopUp > 0 && (earliestTs == 0 || cd.LastTopUp < earliestTs) {
					earliestTs = cd.LastTopUp
				}
				if cd.LastBet > 0 && (earliestTs == 0 || cd.LastBet < earliestTs) {
					earliestTs = cd.LastBet
				}
				if cd.LastWithdrawal > 0 && (earliestTs == 0 || cd.LastWithdrawal < earliestTs) {
					earliestTs = cd.LastWithdrawal
				}
				if earliestTs > 0 {
					cd.CreatedAt = earliestTs
					log.Printf("info: user %s missing createdAt, using first action %d as registration", userId, earliestTs)
				}
			}

			if l, ok := user["login"].(string); ok {
				cd.Login = l
			}
			if f, ok := user["firstName"].(string); ok {
				cd.FirstName = f
			}
			if la, ok := user["lastName"].(string); ok {
				cd.LastName = la
			}
			if ph, ok := user["phone"].(string); ok {
				cd.Phone = ph
			}
			if ci, ok := user["countryId"].(float64); ok {
				cd.CountryId = int(ci)
			}
			if st, ok := user["state"].(float64); ok {
				cd.State = int(st)
			}
		}

		if stats, ok := clientData["stats"].(map[string]interface{}); ok {
			if p, ok := stats["platform"].(float64); ok {
				cd.Platform = int(p)
			}
		}

		if wallets, ok := clientData["wallets"].([]interface{}); ok {
			for _, w := range wallets {
				if wallet, ok := w.(map[string]interface{}); ok {
					if active, ok := wallet["isActive"].(float64); ok && active == 1 {
						switch v := wallet["no"].(type) {
						case string:
							cd.Account.ActiveWallet = v
						case float64:
							cd.Account.ActiveWallet = fmt.Sprintf("%.0f", v)
						}
						if balance, ok := wallet["balance"].(float64); ok {
							cd.Account.Balance = balance
						}
						if currencyId, ok := wallet["currencyId"].(float64); ok {
							cd.Account.CurrencyId = int(currencyId)
						}
						break
					}
				}
			}
		}
	} else {
		cd.CountryId = frontCountryId
		cd.Platform = 0
		cd.Account.CurrencyId = 1

		if cd.LastActivity > 0 {
			var earliestTs int64
			if cd.LastTopUp > 0 && (earliestTs == 0 || cd.LastTopUp < earliestTs) {
				earliestTs = cd.LastTopUp
			}
			if cd.LastBet > 0 && (earliestTs == 0 || cd.LastBet < earliestTs) {
				earliestTs = cd.LastBet
			}
			if cd.LastWithdrawal > 0 && (earliestTs == 0 || cd.LastWithdrawal < earliestTs) {
				earliestTs = cd.LastWithdrawal
			}
			if earliestTs > 0 {
				cd.CreatedAt = earliestTs
				log.Printf("info: orphan user %s - using first action %d as CreatedAt", userId, earliestTs)
			}
		}
	}

	switch frontCountryId {
	case 213:
		cd.Account.CurrencyId = 1
	case 181:
		cd.Account.CurrencyId = 2
	case 233:
		cd.Account.CurrencyId = 3
	}

	if cd.LastTopUp == 0 && cd.LastBet == 0 && cd.LastWithdrawal == 0 {
		cd.LastActivity = 0
	}

	return cd
}
