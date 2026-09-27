package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Environment   string
	HTTPAddr      string
	DatabaseURL   string
	SessionTTL    time.Duration
	DBMaxConns    int32
	DBMinConns    int32
	DBMaxLifetime time.Duration
	DBMaxIdleTime time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Environment: "development", HTTPAddr: ":8080", SessionTTL: 7 * 24 * time.Hour,
		DBMaxConns: 10, DBMinConns: 2, DBMaxLifetime: time.Hour, DBMaxIdleTime: 30 * time.Minute,
	}
	cfg.Environment = env("APP_ENV", cfg.Environment)
	cfg.HTTPAddr = env("HTTP_ADDR", cfg.HTTPAddr)
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	var err error
	if cfg.SessionTTL, err = duration("SESSION_TTL", cfg.SessionTTL); err != nil {
		return Config{}, err
	}
	if cfg.DBMaxLifetime, err = duration("DB_MAX_CONN_LIFETIME", cfg.DBMaxLifetime); err != nil {
		return Config{}, err
	}
	if cfg.DBMaxIdleTime, err = duration("DB_MAX_CONN_IDLE_TIME", cfg.DBMaxIdleTime); err != nil {
		return Config{}, err
	}
	if cfg.DBMaxConns, err = int32Value("DB_MAX_CONNS", cfg.DBMaxConns); err != nil {
		return Config{}, err
	}
	if cfg.DBMinConns, err = int32Value("DB_MIN_CONNS", cfg.DBMinConns); err != nil {
		return Config{}, err
	}
	if cfg.DBMinConns > cfg.DBMaxConns {
		return Config{}, fmt.Errorf("DB_MIN_CONNS cannot exceed DB_MAX_CONNS")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func duration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return d, nil
}
func int32Value(key string, fallback int32) (int32, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return int32(n), nil
}
