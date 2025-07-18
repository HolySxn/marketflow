package service

import (
	"marketflow/internal/core/domain"
	"time"
)

func (s *MarketService) processAggregatedData(aggregator <-chan domain.MarketData) {
	defer s.wg.Done()

	for {
		select {
		case <-s.serviceCtx.Done():
			return
		case data, ok := <-aggregator:
			if !ok {
				s.logger.Info("aggregator data channel closed")
				return
			}

			s.addToBatch(data)
		}
	}
}

func (s *MarketService) calculateAggregates(dataPoints []domain.MarketData) domain.AggregatedData {
	if len(dataPoints) == 0 {
		return domain.AggregatedData{}
	}

	first := dataPoints[0]
	min := first.Price
	max := first.Price
	sum := first.Price

	for i := 1; i < len(dataPoints); i++ {
		price := dataPoints[i].Price
		if price < min {
			min = price
		}
		if price > max {
			max = price
		}
		sum += price
	}

	return domain.AggregatedData{
		Pair:      first.Pair,
		Exchange:  first.Exchange,
		Timestamp: time.Now(),
		Average:   sum / float64(len(dataPoints)),
		Min:       min,
		Max:       max,
	}
}
