package redis

import (
	"context"
	"fmt"
	"log/slog"
	"marketflow/internal/core/domain"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

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

func (c *RCacher) GetLastMinutePrices(ctx context.Context, exchange string, pair string) ([]float64, error) {
	min := time.Now().Unix() - int64(time.Minute.Seconds())
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
