package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/heliocosta10/rate-limiter-go/internal/config"
	"github.com/heliocosta10/rate-limiter-go/internal/limiter"
	"github.com/heliocosta10/rate-limiter-go/internal/middleware"
	"github.com/heliocosta10/rate-limiter-go/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	redisStore := store.NewRedisStore(
		cfg.RedisAddr,
		cfg.RedisPassword,
		cfg.RedisDB,
	)
	defer redisStore.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisStore.Ping(ctx); err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}

	rateLimiter := limiter.New(redisStore, limiter.LimitConfig{
		IPLimit:           cfg.IPLimit,
		DefaultTokenLimit: cfg.DefaultTokenLimit,
		TokenLimits:       cfg.TokenLimits,
		Window:            cfg.Window,
		BlockDuration:     cfg.BlockDuration,
	})

	rateLimitMiddleware := middleware.NewRateLimitMiddleware(rateLimiter)

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("rate limiter is running"))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:            rateLimitMiddleware.Handler(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("rate limiter listening on :%s", cfg.Port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
