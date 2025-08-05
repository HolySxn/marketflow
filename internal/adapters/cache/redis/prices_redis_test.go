package redis

import (
	"context"
	"log/slog"
	"marketflow/internal/core/domain"
	"math/rand"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestAddPrice(t *testing.T) {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to PING Redis: %v", err)
	}

	cacher := NewRedisCache(rdb, slog.Default())

	data := domain.MarketData{
		Exchange:  "testex",
		Pair:      "BTCUSDT",
		Price:     123.45,
		Timestamp: time.Now().Unix(),
	}

	err := cacher.AddPrice(ctx, data)
	if err != nil {
		t.Fatalf("AddPrice failed: %v", err)
	}

	key := "prices:testex:BTCUSDT"
	res, err := rdb.ZRangeWithScores(ctx, key, 0, -1).Result()
	if err != nil {
		t.Fatalf("ZRangeWithScores failed: %v", err)
	}
	if len(res) == 0 {
		t.Fatal("No data found in Redis")
	}
	t.Log(res)

	// Cleanup
	if err := rdb.Del(ctx, key).Err(); err != nil {
		t.Fatalf("Failed to cleanup Redis: %v", err)
	}
}

func TestGetLastMinutePrices(t *testing.T) {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to PING Redis: %v", err)
	}

	cacher := NewRedisCache(rdb, slog.Default())

	min := 100.0
	max := 200.0
	key := "prices:testex:BTCUSDT"

	for i := 0; i < 10; i++ {
		data := domain.MarketData{
			Exchange:  "testex",
			Pair:      "BTCUSDT",
			Price:     min + rand.Float64()*(max-min),
			Timestamp: time.Now().Unix(),
		}

		err := cacher.AddPrice(ctx, data)
		if err != nil {
			t.Fatalf("AddPrice failed: %v", err)
		}
		time.Sleep(time.Second)
	}

	prices, err := cacher.GetPricesByPeriod(ctx, "testex", "BTCUSDT", time.Minute)
	if err != nil {
		t.Fatalf("Failed to get all prices: %v", err)
	}
	if len(prices) != 10 {
		t.Fatalf("Wrong amount of data: %d != 10", len(prices))
	}

	// Cleanup
	if err := rdb.Del(ctx, key).Err(); err != nil {
		t.Fatalf("Failed to cleanup Redis: %v", err)
	}
}

func TestGetLastMinutePricesMoreThanMinute(t *testing.T) {
	ctx := context.Background()

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("Failed to PING Redis: %v", err)
	}

	cacher := NewRedisCache(rdb, slog.Default())

	min := 100.0
	max := 200.0
	key := "prices:testex:BTCUSDT"

	for i := 0; i < 10; i++ {
		timestamp := time.Now().Unix() - int64(time.Minute.Seconds()*2)
		data := domain.MarketData{
			Exchange:  "testex",
			Pair:      "BTCUSDT",
			Price:     min + rand.Float64()*(max-min),
			Timestamp: timestamp,
		}

		err := cacher.AddPrice(ctx, data)
		if err != nil {
			t.Fatalf("AddPrice failed: %v", err)
		}
		time.Sleep(time.Second)
	}

	prices, err := cacher.GetPricesByPeriod(ctx, "testex", "BTCUSDT", time.Minute)
	if err != nil {
		t.Fatalf("Failed to get all prices: %v", err)
	}
	if len(prices) != 0 {
		t.Fatalf("Wrong amount of data: %d != 0", len(prices))
	}

	// Cleanup
	if err := rdb.Del(ctx, key).Err(); err != nil {
		t.Fatalf("Failed to cleanup Redis: %v", err)
	}
}
