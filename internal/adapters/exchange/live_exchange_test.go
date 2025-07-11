package exchange

import (
	"bufio"
	"encoding/json"
	"fmt"
	"marketflow/internal/core/domain"
	"net"
	"testing"
	"time"
)

func startTestTCPServer(t *testing.T, lines []string, delay time.Duration) (addr string, closeFn func()) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				w := bufio.NewWriter(c)
				for _, line := range lines {
					w.WriteString(line + "\n")
					w.Flush()
					if delay > 0 {
						time.Sleep(delay)
					}
				}
			}(conn)
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }
}

func TestLiveExchange_ParsesValidData(t *testing.T) {
	market := domain.MarketData{
		Exchange:  "test",
		Pair:      "BTCUSDT",
		Price:     123.45,
		Timestamp: time.Now().UTC().Truncate(time.Second).Unix(),
	}
	jsonData, _ := json.Marshal(struct {
		Symbol    string  `json:"symbol"`
		Price     float64 `json:"price"`
		Timestamp int64   `json:"timestamp"`
	}{
		Symbol:    market.Pair,
		Price:     market.Price,
		Timestamp: market.Timestamp,
	})

	addr, closeFn := startTestTCPServer(t, []string{string(jsonData)}, 0)
	defer closeFn()

	host, port, _ := net.SplitHostPort(addr)
	e := &LiveExchange{ID: "test", Host: host, Port: port}
	ch := e.Start()

	select {
	case got := <-ch:
		if got.Pair != market.Pair || got.Price != market.Price || got.Timestamp != market.Timestamp {
			t.Errorf("unexpected market data: got %+v, want %+v", got, market)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for market data")
	}
}

func TestLiveExchange_SkipsInvalidJSON(t *testing.T) {
	addr, closeFn := startTestTCPServer(t, []string{"not-json"}, 0)
	defer closeFn()

	host, port, _ := net.SplitHostPort(addr)
	e := &LiveExchange{ID: "test", Host: host, Port: port}
	ch := e.Start()

	select {
	case got, ok := <-ch:
		if ok {
			t.Errorf("expected channel to be closed or empty, got: %+v", got)
		}
	case <-time.After(200 * time.Millisecond):
		// pass: no data received
	}
}

func TestLiveExchange_Reconnects(t *testing.T) {
	// Start a server, send one line, then close. LiveExchange should reconnect.
	json1, _ := json.Marshal(struct {
		Symbol    string    `json:"symbol"`
		Price     float64   `json:"price"`
		Timestamp time.Time `json:"timestamp"`
	}{"BTCUSDT", 1, time.Now().UTC()})
	json2, _ := json.Marshal(struct {
		Symbol    string    `json:"symbol"`
		Price     float64   `json:"price"`
		Timestamp time.Time `json:"timestamp"`
	}{"ETHUSDT", 2, time.Now().UTC()})

	var serverClosed bool
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	addr := ln.Addr().String()
	host, port, _ := net.SplitHostPort(addr)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		w := bufio.NewWriter(conn)
		w.WriteString(string(json1) + "\n")
		w.Flush()
		conn.Close()
		ln.Close()
		serverClosed = true
	}()

	e := &LiveExchange{ID: "test", Host: host, Port: port}
	ch := e.Start()

	// Should get first data
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for first data")
	}

	// Start new server for reconnect
	if !serverClosed {
		t.Fatal("server did not close as expected")
	}
	ln2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("failed to start second server: %v", err)
	}
	defer ln2.Close()
	go func() {
		conn, err := ln2.Accept()
		if err != nil {
			return
		}
		w := bufio.NewWriter(conn)
		w.WriteString(string(json2) + "\n")
		w.Flush()
		conn.Close()
	}()

	// Should get second data after reconnect
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for second data after reconnect")
	}
}

func TestRealExchange(t *testing.T) {
	host, port := "127.0.0.1", "40101"

	e := &LiveExchange{
		ID:   "test",
		Host: host,
		Port: port,
	}
	ch := e.Start()

	count := 0
	max := 5
	timeout := time.After(5 * time.Second)
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				return
			}
			fmt.Println(v)
			count++
			if count >= max {
				return
			}
		case <-timeout:
			t.Fatal("timeout waiting for data")
		}
	}
}
