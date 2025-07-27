package service

import (
	"context"
	"fmt"
	"log/slog"
	"marketflow/internal/core/domain"
	"marketflow/internal/core/port"
	"marketflow/internal/core/service/workerpool"
	"sync"
	"time"
)

type MarketService struct {
	cache      port.CachePort
	repository port.RepositoryPort
	exchanges  []port.ExchangePort
	logger     *slog.Logger

	// Worker pools for each exchange
	workerPools []*workerpool.WorkerPool
	aggregator  *workerpool.FanInAggregator

	// Data batching
	batchMutex  sync.RWMutex
	batchData   map[string][]domain.MarketData
	batchTicker *time.Ticker

	serviceCtx    context.Context
	serviceCancel context.CancelFunc
	wg            sync.WaitGroup
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

// make aggregator
// make worker pool for each exchange and add to aggregator
// read from exchanges and send data to workerpool
// read from aggregator, batch data and aggregate for postgres and push
func (s *MarketService) Start() {
	s.logger.Info("Starting market data service")

	s.workerPools = make([]*workerpool.WorkerPool, len(s.exchanges))
	s.aggregator = workerpool.NewFanInAggregator(s.logger)

	for i := 0; i < len(s.exchanges); i++ {
		pool := workerpool.NewWorkerPool(5, s.cache, s.logger)
		s.workerPools[i] = pool

		outputChan := pool.Start()
		s.aggregator.AddInputChan(outputChan)
	}

	aggregatroChan := s.aggregator.Start()

	s.wg.Add(1)
	go s.processAggregatedData(aggregatroChan)

	for i, ex := range s.exchanges {
		s.wg.Add(1)
		go s.handleExchangeData(ex, s.workerPools[i])
	}

	s.wg.Add(1)
	go s.processBatches()

	s.logger.Info("Market data service started successfully")
}

func (s *MarketService) handleExchangeData(exchange port.ExchangePort, workerpool *workerpool.WorkerPool) {
	defer s.wg.Done()

	dataChan := exchange.Start()

	for {
		select {
		case <-s.serviceCtx.Done():
			return
		case data, ok := <-dataChan:
			if !ok {
				s.logger.Info("Exchange data channel closed")
				return
			}

			workerpool.SubmitJob(data)
		}
	}
}

func (s *MarketService) GetLatestPrice(ctx context.Context, exchange string, pair string) (domain.PriceResponse, error) {
	var cachedPrice *domain.MarketData
	var err error

	if exchange != "" {
		cachedPrice, err = s.cache.GetLatestPrice(ctx, exchange, pair)
	} else {
		cachedPrice, err = s.cache.GetLatestPriceByPair(ctx, pair)
	}

	if err != nil {
		return domain.PriceResponse{}, err
	}

	if cachedPrice != nil {
		return domain.PriceResponse{
			Exchange:  cachedPrice.Exchange,
			Pair:      cachedPrice.Pair,
			Price:     cachedPrice.Price,
			Timestamp: time.Unix(cachedPrice.Timestamp, 0),
		}, nil
	}

	var aggPrice *domain.AggregatedData
	if exchange != "" {
		aggPrice, err = s.repository.GetLatestAggregate(ctx, exchange, pair)
	} else {
		aggPrice, err = s.repository.GetLatestAggregateByPair(ctx, pair)
	}

	if err != nil {
		return domain.PriceResponse{}, err
	}

	if aggPrice != nil {
		return domain.PriceResponse{
			Exchange:  aggPrice.Exchange,
			Pair:      aggPrice.Pair,
			Price:     aggPrice.Average,
			Timestamp: aggPrice.Timestamp,
		}, nil
	}

	return domain.PriceResponse{}, fmt.Errorf("no price found for %s:%s", exchange, pair)
}
