package store

import (
	"context"
	"time"
)

// RateStore is the Strategy interface for persistence.
//
// The limiter only knows this contract, so Redis can be replaced by
// another persistence mechanism without changing the business rule.
type RateStore interface {
	Allow(
		ctx context.Context,
		key string,
		limit int,
		window time.Duration,
		blockDuration time.Duration,
	) (bool, error)
}
