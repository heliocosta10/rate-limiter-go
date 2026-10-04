package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	IPLimit           int
	DefaultTokenLimit int
	TokenLimits       map[string]int
	BlockDuration     time.Duration
	Window            time.Duration
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
}

func Load() (Config, error) {
	// .env is optional; Docker environment variables still work.
	_ = godotenv.Load()

	ipLimit, err := envInt("RATE_LIMIT_IP", 10)
	if err != nil {
		return Config{}, err
	}

	defaultTokenLimit, err := envInt("RATE_LIMIT_TOKEN_DEFAULT", 10)
	if err != nil {
		return Config{}, err
	}

	blockSeconds, err := envInt("BLOCK_DURATION_SECONDS", 300)
	if err != nil {
		return Config{}, err
	}

	windowSeconds, err := envInt("RATE_LIMIT_WINDOW_SECONDS", 1)
	if err != nil {
		return Config{}, err
	}

	redisDB, err := envInt("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}

	tokenLimits := map[string]int{}
	raw := os.Getenv("TOKEN_LIMITS_JSON")
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &tokenLimits); err != nil {
			return Config{}, fmt.Errorf("invalid TOKEN_LIMITS_JSON: %w", err)
		}
	}

	return Config{
		Port:              envString("APP_PORT", "8080"),
		IPLimit:           ipLimit,
		DefaultTokenLimit: defaultTokenLimit,
		TokenLimits:       tokenLimits,
		BlockDuration:     time.Duration(blockSeconds) * time.Second,
		Window:            time.Duration(windowSeconds) * time.Second,
		RedisAddr:         envString("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		RedisDB:           redisDB,
	}, nil
}

func envString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}

	if n < 0 {
		return 0, fmt.Errorf("%s must be zero or greater", key)
	}

	return n, nil
}
