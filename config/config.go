package config

import (
	"os"
	"strconv"
)

type (
	Postgres struct {
		User   string
		Pass   string
		Host   string
		Port   string
		DBName string
	}

	Redis struct {
		Addr string
		DB   int
	}

	ServerConfig struct {
		Port   string
		Host   string
		LogLvl string
	}

	Config struct {
		Postgres Postgres
		Redis    Redis
		Server   ServerConfig
	}
)

func LoadConfig() *Config {
	cfg := &Config{}

	cfg.Postgres.User = getEnv("DB_USER", "postgres")
	cfg.Postgres.Pass = getEnv("DB_PASS", "postgres")
	cfg.Postgres.Host = getEnv("DB_HOST", "localhost")
	cfg.Postgres.Port = getEnv("DB_PORT", "5432")
	cfg.Postgres.DBName = getEnv("DB_NAME", "market")

	cfg.Redis.Addr = getEnv("REDIS_ADDR", "localhost:6379")
	cfg.Redis.DB, _ = strconv.Atoi(getEnv("REDIS_DB", "0"))

	cfg.Server.LogLvl = getEnv("LOG_LVL", "dev")
	cfg.Server.Port = getEnv("PORT", "8080")
	cfg.Server.Host = getEnv("HOST", "0.0.0.0")

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}

	return defaultValue
}
