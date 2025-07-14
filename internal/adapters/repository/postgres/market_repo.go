package postgres

import (
	"context"
	"log/slog"
	"marketflow/internal/core/domain"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MarketRepository struct {
	db     *pgxpool.Pool
	logger *slog.Logger
}

func NewMarketRepository(db *pgxpool.Pool, logger *slog.Logger) *MarketRepository {
	return &MarketRepository{
		db:     db,
		logger: logger,
	}
}

func (r *MarketRepository) SaveAggregate(ctx context.Context, data domain.AggregatedData) error {
	query := `
		INSERT INTO market_aggregates (pair_name, exchange, timestamp, average_price, min_price, max_price)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		data.Pair,
		data.Exchange,
		data.Timestamp,
		data.Average,
		data.Min,
		data.Max,
	)

	if err != nil {
		r.logger.Error("failed to save aggregate", slog.Any("error", err))
		return err
	}

	return nil
}

func (r *MarketRepository) GetAggregatesByPeriod(ctx context.Context, exchange string, pair string, period time.Duration) ([]domain.AggregatedData, error) {
	query := `
		SELECT pair_name, exchange, timestamp, average_price, min_price, max_price
		FROM market_aggregates
		WHERE exchange = $1 AND pair_name = $2 AND timestamp > $3
		ORDER BY timestamp DESC
	`

	since := time.Now().Add(-period)
	rows, err := r.db.Query(ctx, query, exchange, pair, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []domain.AggregatedData
	for rows.Next() {
		var agg domain.AggregatedData
		err := rows.Scan(
			&agg.Pair,
			&agg.Exchange,
			&agg.Timestamp,
			&agg.Average,
			&agg.Min,
			&agg.Max,
		)
		if err != nil {
			r.logger.Error("failed to scan aggregate", slog.Any("error", err))
			continue
		}
		results = append(results, agg)
	}

	return results, nil
}

func (r *MarketRepository) GetLatestAggregate(ctx context.Context, exchange string, pair string) (domain.AggregatedData, error) {
	query := `
		SELECT pair_name, exchange, timestamp, average_price, min_price, max_price
		FROM market_aggregates
		WHERE exchange = $1 AND pair_name = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var agg domain.AggregatedData
	err := r.db.QueryRow(ctx, query, exchange, pair).Scan(
		&agg.Pair,
		&agg.Exchange,
		&agg.Timestamp,
		&agg.Average,
		&agg.Min,
		&agg.Max,
	)

	if err != nil {
		return domain.AggregatedData{}, err
	}

	return agg, nil
}
