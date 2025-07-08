package domain

import "time"

type MarketData struct {
	Exchange  string    `json:"exchange,omitempty"` // Exchange name/source
	Pair      string    `json:"symbol"`             // "BTCUSDT", "ETHUSDT", etc.
	Price     float64   `json:"price"`              // Current price
	Timestamp time.Time `json:"timestamp"`          // When the data was received
}
