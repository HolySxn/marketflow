package exchange

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"marketflow/internal/core/domain"
	"marketflow/internal/core/port"
	"net"
	"sync"
	"time"
)

var _ port.ExchangePort = (*LiveExchange)(nil)

const reconnectDelay = 5 * time.Second

type LiveExchange struct {
	ID       string
	Host     string
	Port     string
	Logger   *slog.Logger
	conn     net.Conn
	dataChan chan domain.MarketData
	stopChan chan struct{}

	wg sync.WaitGroup
	mu sync.RWMutex

	connected bool
}

func NewLiveExchange(id, host, port string, logger *slog.Logger) *LiveExchange {
	return &LiveExchange{
		ID:       id,
		Host:     host,
		Port:     port,
		Logger:   logger,
		dataChan: make(chan domain.MarketData),
		stopChan: make(chan struct{}),
	}
}

// Start initiates the connection management loop and returns the data channel.
func (e *LiveExchange) Start() <-chan domain.MarketData {
	e.wg.Add(1)
	go e.manageConnection()
	return e.dataChan
}

// Stop signals the connection management loop to terminate and closes the connection.
func (e *LiveExchange) Stop() {
	e.Logger.Info("stopping exchange adapter")
	close(e.stopChan)
	e.mu.Lock()
	if e.conn != nil {
		e.conn.Close()
	}
	e.mu.Unlock()
	e.wg.Wait()
	close(e.dataChan)
	e.Logger.Info("exchange adapter is closed")
}

func (e *LiveExchange) IsConnected() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.connected
}

func (e *LiveExchange) setConnected(status bool) {
	e.mu.Lock()
	e.connected = status
	e.mu.Unlock()
}

func (e *LiveExchange) manageConnection() {
	defer e.wg.Done()
	e.Logger.Info("starting connection manager")

	for {
		select {
		case <-e.stopChan:
			e.Logger.Info("shutting down connection manager")
			return
		default:
			if !e.connected {
				if err := e.connect(); err != nil {
					e.Logger.Error("failed to connect", "error", err, "retry_in", reconnectDelay)
					time.Sleep(reconnectDelay)
					continue
				}
			}

			err := e.readData()
			e.setConnected(false)
			if err != io.EOF {
				e.Logger.Warn("connection lost", "error", err)
			} else {
				e.Logger.Info("connection closed by remote")
			}
		}
	}
}

func (e *LiveExchange) connect() error {
	e.Logger.Info("attempting to connect")
	addr := net.JoinHostPort(e.Host, e.Port)

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}

	e.mu.Lock()
	e.conn = conn
	e.mu.Unlock()

	e.setConnected(true)
	e.Logger.Info("successfully connected")
	return nil
}

func (e *LiveExchange) readData() error {
	e.Logger.Info("starting to read data from connection")
	reader := bufio.NewReader(e.conn)

	for {
		select {
		case <-e.stopChan:
			return nil
		default:
			e.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if nerErr, ok := err.(net.Error); ok && nerErr.Timeout() {
					e.Logger.Info("read timeout, will try again")
					continue
				}

				e.Logger.Error("read error", "error", err)
				e.conn.Close()
				return err
			}

			var data domain.MarketData
			if err := json.Unmarshal(line, &data); err != nil {
				e.Logger.Warn("failed to unmarshal data", "data", string(line), "error", err)
				continue
			}

			data.Exchange = e.ID

			e.dataChan <- data
		}
	}
}
