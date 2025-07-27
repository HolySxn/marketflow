package redis

import (
	"context"
	"fmt"
	"log/slog"
	"marketflow/internal/core/domain"
	"marketflow/internal/core/port"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var _ port.CachePort = (*RCacher)(nil)

type RCacher struct {
	rdb    *redis.Client
	logger *slog.Logger
}

func NewRCacher(db *redis.Client, logger *slog.Logger) *RCacher {
	return &RCacher{
		rdb:    db,
		logger: logger,
	}
}

func (c *RCacher) AddPrice(ctx context.Context, data domain.MarketData) error {
	err := c.rdb.ZAdd(
		ctx,
		fmt.Sprintf("prices:%s:%s", data.Exchange, data.Pair),
		redis.Z{
			Member: data.Price,
			Score:  float64(data.Timestamp),
		},
	).Err()

	if err != nil {
		return err
	}

	return nil
}

func (c *RCacher) GetPricesByPeriod(ctx context.Context, exchange string, pair string, period time.Duration) ([]float64, error) {
	min := time.Now().Add(-period).Unix()
	max := time.Now().Unix()
	key := fmt.Sprintf("prices:%s:%s", exchange, pair)

	res, err := c.rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", min),
		Max: fmt.Sprintf("%d", max),
	}).Result()
	if err != nil {
		return nil, err
	}

	prices := make([]float64, 0, len(res))
	for _, v := range res {
		price, err := strconv.ParseFloat(v, 64)
		if err != nil {
			c.logger.Error("failed to parse price", slog.Any("value", v), slog.Any("err", err))
			continue
		}
		prices = append(prices, price)
	}

	return prices, nil
}

func (c *RCacher) GetLatestPrice(ctx context.Context, exchange string, pair string) (*domain.MarketData, error) {
	key := fmt.Sprintf("prices:%s:%s", exchange, pair)

	res, err := c.rdb.ZRevRangeWithScores(ctx, key, 0, 0).Result()
	if err != nil || len(res) == 0 {
		return nil, fmt.Errorf("no price found for %s:%s", exchange, pair)
	}

	price := res[0].Member.(string)
	score := res[0].Score
	p, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return nil, err
	}

	return &domain.MarketData{
		Exchange:  exchange,
		Pair:      pair,
		Price:     p,
		Timestamp: int64(score),
	}, nil
}

func (c *RCacher) GetLatestPriceByPair(ctx context.Context, pair string) (*domain.MarketData, error) {
	pattern := fmt.Sprintf("prices:*:%s", pair)
	var latest *domain.MarketData
	var latestScore float64

	iter := c.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		res, err := c.rdb.ZRevRangeWithScores(ctx, key, 0, 0).Result()
		if err != nil || len(res) == 0 {
			continue
		}
		price := res[0].Member.(string)
		score := res[0].Score
		p, err := strconv.ParseFloat(price, 64)
		if err != nil {
			continue
		}
		if latest == nil || score > latestScore {
			latest = &domain.MarketData{
				Exchange:  extractExchangeFromKey(key),
				Pair:      pair,
				Price:     p,
				Timestamp: int64(score),
			}
			latestScore = score
		}
	}
	if latest == nil {
		return nil, fmt.Errorf("no price found for symbol %s", pair)
	}
	return latest, nil
}

func extractExchangeFromKey(key string) string {
	// key format: prices:<exchange>:<pair>
	parts := strings.Split(key, ":")
	if len(parts) >= 3 {
		return parts[1]
	}
	return ""
}
