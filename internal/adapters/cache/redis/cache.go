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

var _ port.CachePort = (*RedisCache)(nil)

const (
	priceTTL = 5 * time.Minute
)

type RedisCache struct {
	client *redis.Client
	logger *slog.Logger
}

func NewRedisCache(client *redis.Client, logger *slog.Logger) *RedisCache {
	return &RedisCache{
		client: client,
		logger: logger,
	}
}

// Ping checks the connection to the Redis server.
func (c *RedisCache) Ping(ctx context.Context) string {
	if err := c.client.Ping(ctx).Err(); err != nil {
		return fmt.Sprintf("down: %v", err)
	}
	return "up"
}

// keyForExchange returns the Redis key for a specific exchange and pair.
func (c *RedisCache) keyForExchange(exchange, pair string) string {
	return fmt.Sprintf("prices:%s:%s", exchange, pair)
}

// keyForPair returns the Redis key for a pair across all exchanges.
func (c *RedisCache) keyForPair(pair string) string {
	return fmt.Sprintf("prices:all:%s", pair)
}

// AddPrice adds a new price point to the cache. It stores data in two
// sorted sets: one for the specific exchange+pair and one for the pair across
// all exchanges. This allows for efficient lookups in both scenarios.
// It also cleans up old data points to prevent the cache from growing indefinitely.
func (c *RedisCache) AddPrice(ctx context.Context, data domain.MarketData) error {
	exchangeKey := c.keyForExchange(data.Exchange, data.Pair)
	pairKey := c.keyForPair(data.Pair)

	timestampScore := float64(data.Timestamp)
	priceMember := strconv.FormatFloat(data.Price, 'f', -1, 64)
	pairMember := fmt.Sprintf("%s:%s", data.Exchange, priceMember)

	// Use Pipeline to execute commands efficiently
	pipe := c.client.Pipeline()

	// Add the new price
	pipe.ZAdd(ctx, exchangeKey, redis.Z{Score: timestampScore, Member: priceMember})
	pipe.ZAdd(ctx, pairKey, redis.Z{Score: timestampScore, Member: pairMember})

	// Remove old entries
	maxScore := float64(time.Now().Add(-priceTTL).Unix())
	pipe.ZRemRangeByScore(ctx, exchangeKey, "-inf", fmt.Sprintf("(%f", maxScore))
	pipe.ZRemRangeByScore(ctx, pairKey, "-inf", fmt.Sprintf("(%f", maxScore))

	// Set an expiration on the keys themselves, in case a pair stops receiving updates.
	pipe.Expire(ctx, exchangeKey, priceTTL)
	pipe.Expire(ctx, pairKey, priceTTL)

	_, err := pipe.Exec(ctx)
	if err != nil {
		c.logger.Error("failed to add price to redis cache", "error", err)
		return err
	}

	return nil
}

// GetPricesByPeriod retrieves all price points for a given exchange and pair within a specified duration.
func (c *RedisCache) GetPricesByPeriod(ctx context.Context, exchange string, pair string, period time.Duration) ([]float64, error) {
	key := c.keyForExchange(exchange, pair)
	minScore := time.Now().Add(-period).Unix()

	members, err := c.client.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min: fmt.Sprintf("%d", minScore),
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, err
	}

	prices := make([]float64, 0, len(members))
	for _, member := range members {
		price, err := strconv.ParseFloat(member, 64)
		if err != nil {
			c.logger.Warn("could not parse price from redis member", "member", member, "error", err)
			continue
		}
		prices = append(prices, price)
	}

	return prices, nil
}

// GetLatestPrice retrieves the most recent price for a specific exchange and pair.
func (c *RedisCache) GetLatestPrice(ctx context.Context, exchange string, pair string) (*domain.MarketData, error) {
	key := c.keyForExchange(exchange, pair)

	result, err := c.client.ZRevRangeWithScores(ctx, key, 0, 0).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}

	price, err := strconv.ParseFloat(result[0].Member.(string), 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse latest price from member: %w", err)
	}

	return &domain.MarketData{
		Exchange:  exchange,
		Pair:      pair,
		Price:     price,
		Timestamp: int64(price),
	}, nil
}

// GetLatestPriceByPair retrieves the most recent price for a pair from any exchange.
func (c *RedisCache) GetLatestPriceByPair(ctx context.Context, pair string) (*domain.MarketData, error) {
	key := c.keyForPair(pair)

	result, err := c.client.ZRevRangeWithScores(ctx, key, 0, 0).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}

	member := result[0].Member.(string)
	parts := strings.SplitN(member, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid member format for pair price: %s", member)
	}
	exchange := parts[0]
	price, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse latest pair price from member: %w", err)
	}

	return &domain.MarketData{
		Exchange:  exchange,
		Pair:      pair,
		Price:     price,
		Timestamp: int64(result[0].Score),
	}, nil
}
