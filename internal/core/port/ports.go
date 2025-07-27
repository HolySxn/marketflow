package port

import (
	"context"
	"marketflow/internal/core/domain"
	"time"
)

type CachePort interface {
	AddPrice(ctx context.Context, data domain.MarketData) error
	GetPricesByPeriod(ctx context.Context, exchange string, pair string, period time.Duration) ([]float64, error)
	GetLatestPrice(ctx context.Context, exchange string, pair string) (*domain.MarketData, error)
	GetLatestPriceByPair(ctx context.Context, pair string) (*domain.MarketData, error)
}

type RepositoryPort interface {
	SaveAggregate(ctx context.Context, data domain.AggregatedData) error
	GetAggregatesByPeriod(ctx context.Context, exchange string, pair string, period time.Duration) ([]domain.AggregatedData, error)
	GetLatestAggregate(ctx context.Context, exchange string, pair string) (*domain.AggregatedData, error)
	GetLatestAggregateByPair(ctx context.Context, pair string) (*domain.AggregatedData, error)
}

type ExchangePort interface {
	Start() <-chan domain.MarketData
	Stop()
	IsConnected() bool
}

// MarketServicePort defines market service operations
type MarketServicePort interface {
	Start()
	Stop()
	GetLatestPrice(ctx context.Context, exchange string, pair string) (domain.PriceResponse, error)
	GetHighestPrice(ctx context.Context, exchange string, pair string, period time.Duration) (domain.PriceResponse, error)
	GetLowestPrice(ctx context.Context, exchange string, pair string, period time.Duration) (domain.PriceResponse, error)
	GetAveragePrice(ctx context.Context, exchange string, pair string, period time.Duration) (domain.PriceResponse, error)
	SwitchToTestMode() error
	SwitchToLiveMode() error
	GetHealthStatus() domain.HealthResponse
}
