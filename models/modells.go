package models

import _ "time"

// Hit представляет структуру одного результата поиска.
type Hit struct {
	Index  string                 `json:"_index"`
	Source map[string]interface{} `json:"_source"`
}

// SearchResponse представляет ответ поискового запроса.
type SearchResponse struct {
	Hits struct {
		Hits []Hit `json:"hits"`
	} `json:"hits"`
}

// Account представляет данные аккаунта клиента.
type Account struct {
	ActiveWallet string  `json:"activeWallet"`
	Balance      float64 `json:"balance"`
	CurrencyId   int     `json:"currencyId"`
}

// ClientData представляет данные клиента.
// Добавлено поле CanReactivate для удобства фронта.
type ClientData struct {
	Platform              int    `json:"platform"`
	CreatedAt             int64  `json:"createdAt"`
	Login                 string `json:"login"`
	FirstName             string `json:"firstName"`
	LastName              string `json:"lastName"`
	Phone                 string `json:"phone"`
	Account               Account
	CountryId             int    `json:"countryId"`
	State                 int    `json:"state"`
	LastTopUp             int64  `json:"lastTopUp"`      // unix
	LastBet               int64  `json:"lastBet"`        // unix
	LastWithdrawal        int64  `json:"lastWithdrawal"` // unix
	UserId                string `json:"userId"`
	LastActivity          int64  `json:"lastActivity"`
	ReactivationThreshold int64  `json:"reactivationThreshold"` // unix timestamp порога (lastActivity - months)
	CanReactivate         bool   `json:"canReactivate"`         // можно ли реактивировать сейчас
}
