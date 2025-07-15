package workerpool

import (
	"context"
	"log/slog"
	"marketflow/internal/core/domain"
	"marketflow/internal/core/port"
	"sync"
)

type WorkerPool struct {
	maxWorkers int
	jobQueue   chan domain.MarketData
	outputChan chan domain.MarketData
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	stopOnce   sync.Once

	cacher port.CachePort
	logger *slog.Logger
}

func NewWorkerPool(maxWorkers int, cacher port.CachePort, logger *slog.Logger) *WorkerPool {
	if maxWorkers < 1 {
		maxWorkers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &WorkerPool{
		maxWorkers: maxWorkers,
		jobQueue:   make(chan domain.MarketData, 1000),
		outputChan: make(chan domain.MarketData, 1000),
		ctx:        ctx,
		cancel:     cancel,
		cacher:     cacher,
		logger:     logger,
	}

	return pool
}

// Submit enqueues a function for a worker to execute.
func (p *WorkerPool) SubmitJob(data domain.MarketData) {
	select {
	case p.jobQueue <- data:
	case <-p.ctx.Done():
		return
	default:
		// Queue is full, drop task
	}
}

// Stop stops the workerpool and waits
func (p *WorkerPool) Stop() {
	p.stopOnce.Do(func() {
		p.cancel()

		close(p.jobQueue)
		p.wg.Wait()
		close(p.jobQueue)
	})
}

func (p *WorkerPool) Start() <-chan domain.MarketData {
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker()
	}

	return p.outputChan
}

func (p *WorkerPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case data, ok := <-p.jobQueue:
			if !ok {
				return
			}

			processedData := p.processMarketData(data)

			select {
			case p.outputChan <- processedData:
			case <-p.ctx.Done():
				return
			}

		}
	}
}

func (p *WorkerPool) processMarketData(data domain.MarketData) domain.MarketData {
	if err := p.cacher.AddPrice(p.ctx, data); err != nil {
		p.logger.Warn("redis add‑price failed", slog.Any("err", err))
	}

	return data
}
