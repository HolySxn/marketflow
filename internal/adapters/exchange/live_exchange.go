package exchange

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"marketflow/internal/core/domain"
	"net"
	"sync"
	"time"
)

type LiveExchange struct {
	ID        string
	Host      string
	Port      string
	outChan   chan domain.MarketData
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	connected bool
	mu        sync.RWMutex
	logger    *slog.Logger
}

type rawMarketData struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
}

func NewLiveExchange(id string, host string, port string, logger *slog.Logger) *LiveExchange {
	ctx, cancel := context.WithCancel(context.Background())
	return &LiveExchange{
		ID:     id,
		Host:   host,
		Port:   port,
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}
}

func (e *LiveExchange) Start() <-chan domain.MarketData {
	e.outChan = make(chan domain.MarketData, 100)

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		defer close(e.outChan)

		for {
			select {
			case <-e.ctx.Done():
				return
			default:
				e.connectAndRead()
			}
		}
	}()

	return e.outChan
}

func (e *LiveExchange) Stop() {
	e.cancel()
	e.wg.Wait()
}

func (e *LiveExchange) IsConnected() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.connected
}

func (e *LiveExchange) connectAndRead() {
	conn, err := net.Dial("tcp", net.JoinHostPort(e.Host, e.Port))
	if err != nil {
		e.setConnected(false)
		e.logger.Error("failed to connect to exchange",
			slog.String("exchange", e.ID),
			slog.String("address", net.JoinHostPort(e.Host, e.Port)),
			slog.Any("error", err))

		select {
		case <-e.ctx.Done():
			return
		case <-time.After(5 * time.Second):
			// Retry connection
		}
		return
	}

	e.setConnected(true)
	e.logger.Info("connected to exchange",
		slog.String("exchange", e.ID),
		slog.String("address", net.JoinHostPort(e.Host, e.Port)))

	e.readStream(conn)
}

func (e *LiveExchange) readStream(conn net.Conn) {
	defer conn.Close()
	defer e.setConnected(false)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		select {
		case <-e.ctx.Done():
			return
		default:
			data, err := parseData(scanner.Text(), e.ID)
			if err != nil {
				e.logger.Error("parse error",
					slog.String("exchange", e.ID),
					slog.Any("error", err))

				continue
			}

			select {
			case e.outChan <- data:
			case <-e.ctx.Done():
				return
			}
		}
	}
}

func (e *LiveExchange) setConnected(connected bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.connected = connected
}

func parseData(data string, exchangeID string) (domain.MarketData, error) {
	var raw rawMarketData

	err := json.Unmarshal([]byte(data), &raw)
	if err != nil {
		return domain.MarketData{}, fmt.Errorf("json unmarshal error: %w", err)
	}

	return domain.MarketData{
		Exchange:  exchangeID,
		Pair:      raw.Symbol,
		Price:     raw.Price,
		Timestamp: raw.Timestamp,
	}, nil
}
