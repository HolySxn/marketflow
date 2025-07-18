package service

import (
	"log/slog"
	"marketflow/internal/core/domain"
)

func (s *MarketService) processBatches() {
	defer s.wg.Done()

	for {
		select {
		case <-s.serviceCtx.Done():
			return
		case <-s.batchTicker.C:
			s.processBatch()
		}
	}
}

func (s *MarketService) processBatch() {
	s.batchMutex.Lock()
	currentBatch := s.batchData
	s.batchData = make(map[string][]domain.MarketData)
	s.batchMutex.Unlock()

	for key, dataPoints := range currentBatch {
		if len(dataPoints) == 0 {
			continue
		}

		aggregate := s.calculateAggregates(dataPoints)

		if err := s.repository.SaveAggregate(s.serviceCtx, aggregate); err != nil {
			s.logger.Error("Failed to save aggregate",
				slog.String("key", key),
				slog.Any("error", err))
		}
	}
}

func (s *MarketService) addToBatch(data domain.MarketData) {
	s.batchMutex.Lock()
	key := data.Exchange + ":" + data.Pair
	s.batchData[key] = append(s.batchData[key], data)
	s.batchMutex.Unlock()
}
