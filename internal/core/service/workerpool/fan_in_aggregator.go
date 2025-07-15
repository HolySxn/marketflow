package workerpool

import (
	"context"
	"log/slog"
	"marketflow/internal/core/domain"
	"sync"
)

type FanInAggregator struct {
	inputChannels []<-chan domain.MarketData
	outputChan    chan domain.MarketData
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	logger        *slog.Logger
}

func NewFanInAggregator(logger *slog.Logger) *FanInAggregator {
	ctx, cancel := context.WithCancel(context.Background())

	return &FanInAggregator{
		outputChan: make(chan domain.MarketData, 1000),
		ctx:        ctx,
		cancel:     cancel,
		logger:     logger,
	}
}

func (f *FanInAggregator) AddInputChan(ch <-chan domain.MarketData) {
	f.inputChannels = append(f.inputChannels, ch)
}

func (f *FanInAggregator) Start() <-chan domain.MarketData {
	f.logger.Info("Starting fan-in aggregator",
		slog.Int("input_channels", len(f.inputChannels)))

	for i, inputChan := range f.inputChannels {
		f.wg.Add(1)
		go f.aggregateFrom(inputChan, i)
	}

	go func() {
		f.wg.Wait()
		close(f.outputChan)
		f.logger.Info("Fan-in aggregator stopped")
	}()

	return f.outputChan
}

func (f *FanInAggregator) Stop() {
	f.logger.Info("Stopping fan-in aggregator")
	f.cancel()
	f.wg.Wait()
}

func (f *FanInAggregator) aggregateFrom(input <-chan domain.MarketData, channelID int) {
	defer f.wg.Done()

	f.logger.Debug("Starting aggregation from channel",
		slog.Int("channel_id", channelID))

	for {
		select {
		case <-f.ctx.Done():
			f.logger.Debug("Aggregation stopped by context",
				slog.Int("channel_id", channelID))
			return
		case data, ok := <-input:
			if !ok {
				f.logger.Debug("Input channel closed",
					slog.Int("channel_id", channelID))
				return
			}

			select {
			case f.outputChan <- data:
				f.logger.Debug("Data aggregated",
					slog.Int("channel_id", channelID),
					slog.String("exchange", data.Exchange),
					slog.String("pair", data.Pair))
			case <-f.ctx.Done():
				return
			}
		}
	}
}
