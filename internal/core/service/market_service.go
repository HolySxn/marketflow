package service

import (
	"context"
	"log/slog"
	"marketflow/internal/core/domain"
	"marketflow/internal/core/port"
	"marketflow/internal/core/service/workerpool"
	"sync"
	"time"
)

// get data from source
// save it in redis
// get data for last minute, calculate average price for each pair and store in postgres with min and max

type MarketService struct {
	cache      port.CachePort
	repository port.RepositoryPort
	exchanges  []port.ExchangePort
	logger     *slog.Logger

	// Worker pools for each exchange
	workerPools []*workerpool.WorkerPool
	//aggregator  *workerpool.FanInAggregator

	// Data batching
	batchMutex  sync.RWMutex
	batchData   map[string][]domain.MarketData
	batchTicker *time.Ticker

	serviceCtx    context.Context
	serviceCancel context.CancelFunc
}

func NewMarketService(
	cache port.CachePort,
	repository port.RepositoryPort,
	exchanges []port.ExchangePort,
	logger *slog.Logger,
) *MarketService {
	serviceCtx, serviceCancel := context.WithCancel(context.Background())

	return &MarketService{
		cache:         cache,
		repository:    repository,
		exchanges:     exchanges,
		logger:        logger,
		batchData:     make(map[string][]domain.MarketData),
		batchTicker:   time.NewTicker(1 * time.Minute),
		serviceCtx:    serviceCtx,
		serviceCancel: serviceCancel,
	}
}

func (s *MarketService) Start() {
	s.logger.Info("Starting market data service")
}
