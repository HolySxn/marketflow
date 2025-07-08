package domain

import "time"

type AggregatedData struct {
	Pair      string
	Exchange  string
	Timestamp time.Time
	Average   float64
	Min       float64
	Max       float64
}
