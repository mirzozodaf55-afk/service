package models

import _ "time"

type Hit struct {
	Index  string                 `json:"_index"`
	Source map[string]interface{} `json:"_source"`
}

type SearchResponse struct {
	Hits struct {
		Hits []Hit `json:"hits"`
	} `json:"hits"`
}

type Account struct {
	ActiveWallet string  `json:"activeWallet"`
	Balance      float64 `json:"balance"`
	CurrencyId   int     `json:"currencyId"`
}

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
	LastTopUp             int64  `json:"lastTopUp"`
	LastBet               int64  `json:"lastBet"`
	LastWithdrawal        int64  `json:"lastWithdrawal"`
	UserId                string `json:"userId"`
	LastActivity          int64  `json:"lastActivity"`
	ReactivationThreshold int64  `json:"reactivationThreshold"`
	CanReactivate         bool   `json:"canReactivate"`
}
