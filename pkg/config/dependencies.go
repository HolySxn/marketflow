package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Dependencies struct {
	Postgres *pgxpool.Pool
	Redis    *redis.Client
	Logger   *slog.Logger
}

type Option func(context.Context, *Dependencies) error

func (d *Dependencies) Close() {
	if d == nil {
		return
	}

	if d.Postgres != nil {
		d.Postgres.Close()
	}
	if d.Redis != nil {
		d.Redis.Close()
	}
}

func NewDependencies(ctx context.Context, opts ...Option) (deps *Dependencies, err error) {
	defer func() {
		if err != nil {
			deps.Close()
		}
	}()

	deps = &Dependencies{}

	for _, opt := range opts {
		if err := opt(ctx, deps); err != nil {
			return nil, err
		}
	}

	return deps, nil
}

func WithPostgres(
	user string,
	password string,
	host string,
	port string,
	dbName string,
) Option {
	return func(ctx context.Context, d *Dependencies) error {
		format := "postgresql://%s:%s@%s:%s/%s?sslmode=disable"
		connString := fmt.Sprintf(format, user, password, host, port, dbName)

		pool, err := pgxpool.New(ctx, connString)
		if err != nil {
			return err
		}

		d.Postgres = pool
		return nil
	}
}

func WithRedis(addr string, db int) Option {
	return func(ctx context.Context, d *Dependencies) error {
		client := redis.NewClient(&redis.Options{
			Addr: addr,
			DB:   db,
		})

		if err := client.Ping(ctx).Err(); err != nil {
			return err
		}

		d.Redis = client
		return nil
	}
}

const (
	EnvDev  = "dev"
	EnvProd = "prod"
)

func WithLogger(level string) Option {
	return func(_ context.Context, d *Dependencies) error {
		var logLvl slog.Level

		switch level {
		case EnvDev:
			logLvl = slog.LevelDebug
		case EnvProd:
			logLvl = slog.LevelInfo
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLvl,
		}))
		slog.SetDefault(logger)
		d.Logger = logger
		return nil
	}
}
