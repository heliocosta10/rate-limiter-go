package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/heliocosta10/rate-limiter-go/internal/limiter"
)

type fakeStore struct {
	allowed bool
	calls   []call
}

type call struct {
	key   string
	limit int
}

func (f *fakeStore) Allow(
	_ context.Context,
	key string,
	limit int,
	_ time.Duration,
	_ time.Duration,
) (bool, error) {
	f.calls = append(f.calls, call{key: key, limit: limit})
	return f.allowed, nil
}

func TestReturns429WithExactMessage(t *testing.T) {
	store := &fakeStore{allowed: false}

	l := limiter.New(store, limiter.LimitConfig{
		IPLimit:           10,
		DefaultTokenLimit: 20,
		TokenLimits:       map[string]int{},
		Window:            time.Second,
		BlockDuration:     time.Minute,
	})

	middleware := NewRateLimitMiddleware(l)
	handler := middleware.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}

	if rec.Body.String() != blockedMessage {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestTokenOverridesIP(t *testing.T) {
	store := &fakeStore{allowed: true}

	l := limiter.New(store, limiter.LimitConfig{
		IPLimit:           10,
		DefaultTokenLimit: 20,
		TokenLimits:       map[string]int{"premium": 100},
		Window:            time.Second,
		BlockDuration:     time.Minute,
	})

	handler := NewRateLimitMiddleware(l).Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("API_KEY", "premium")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if len(store.calls) != 1 {
		t.Fatalf("expected one persistence call, got %d", len(store.calls))
	}

	if store.calls[0].key != "token:premium" {
		t.Fatalf("expected token key, got %s", store.calls[0].key)
	}

	if store.calls[0].limit != 100 {
		t.Fatalf("expected token limit 100, got %d", store.calls[0].limit)
	}
}
