package domain

import "time"

type MarketData struct {
	Exchange  string  `json:"exchange,omitempty"`
	Pair      string  `json:"symbol"`
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
}

type AggregatedData struct {
	Pair      string
	Exchange  string
	Timestamp time.Time
	Average   float64
	Min       float64
	Max       float64
}

type PriceResponse struct {
	Exchange  string    `json:"exchange,omitempty"`
	Pair      string    `json:"pair"`
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Redis     string `json:"redis"`
	Postgres  string `json:"postgres"`
	Exchanges string `json:"exchanges"`
}
