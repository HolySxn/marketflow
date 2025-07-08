package exchange

import (
	"bufio"
	"encoding/json"
	"fmt"
	"marketflow/internal/core/domain"
	"net"
	"time"
)

type LiveExchange struct {
	ID      string
	Host    string
	Port    string
	outChan chan domain.MarketData
}

type rawMarketData struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
}

func (e *LiveExchange) Start() <-chan domain.MarketData {
	e.outChan = make(chan domain.MarketData, 100)

	go func() {
		defer close(e.outChan)

		for {
			conn, err := net.Dial("tcp", net.JoinHostPort(e.Host, e.Port))
			if err != nil {
				time.Sleep(5 * time.Second)
				continue
			}

			e.readStream(conn)
		}
	}()

	return e.outChan
}

func (e *LiveExchange) readStream(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		data, err := parseData(scanner.Text(), e.ID)
		if err != nil {
			fmt.Printf("parse error: %v\n", err)
			continue
		}
		e.outChan <- data
	}
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
		Timestamp: time.Unix(0, raw.Timestamp*int64(time.Millisecond)),
	}, nil
}
