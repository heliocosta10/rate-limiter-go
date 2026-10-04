package limiter

import (
	"context"
	"time"
)

type IdentityType string

const (
	IdentityIP    IdentityType = "ip"
	IdentityToken IdentityType = "token"
)

type Identity struct {
	Type  IdentityType
	Value string
}

type LimitConfig struct {
	IPLimit           int
	DefaultTokenLimit int
	TokenLimits       map[string]int
	Window            time.Duration
	BlockDuration     time.Duration
}

type RateStore interface {
	Allow(
		ctx context.Context,
		key string,
		limit int,
		window time.Duration,
		blockDuration time.Duration,
	) (bool, error)
}

type RateLimiter struct {
	store  RateStore
	config LimitConfig
}

func New(store RateStore, config LimitConfig) *RateLimiter {
	return &RateLimiter{
		store:  store,
		config: config,
	}
}

// Allow applies the business rule independently of HTTP.
//
// Token has precedence over IP: when a token exists, only the token bucket
// is evaluated. The IP bucket is not evaluated for that request.
func (r *RateLimiter) Allow(ctx context.Context, identity Identity) (bool, error) {
	limit := r.config.IPLimit

	if identity.Type == IdentityToken {
		limit = r.config.DefaultTokenLimit

		if configured, ok := r.config.TokenLimits[identity.Value]; ok {
			limit = configured
		}
	}

	key := string(identity.Type) + ":" + identity.Value

	return r.store.Allow(
		ctx,
		key,
		limit,
		r.config.Window,
		r.config.BlockDuration,
	)
}
