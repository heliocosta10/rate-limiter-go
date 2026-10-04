package limiter

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeStore struct {
	mu       sync.Mutex
	calls    []fakeCall
	allowed  bool
}

type fakeCall struct {
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
	f.mu.Lock()
	defer f.mu.Unlock()

	f.calls = append(f.calls, fakeCall{key: key, limit: limit})
	return f.allowed, nil
}

func TestTokenHasPrecedenceOverIP(t *testing.T) {
	store := &fakeStore{allowed: true}

	l := New(store, LimitConfig{
		IPLimit:           10,
		DefaultTokenLimit: 20,
		TokenLimits:       map[string]int{"premium": 100},
		Window:            time.Second,
		BlockDuration:     5 * time.Minute,
	})

	ok, err := l.Allow(context.Background(), Identity{
		Type:  IdentityToken,
		Value: "premium",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected request to be allowed")
	}

	if len(store.calls) != 1 {
		t.Fatalf("expected one store call, got %d", len(store.calls))
	}

	call := store.calls[0]
	if call.key != "token:premium" {
		t.Fatalf("expected token key, got %s", call.key)
	}
	if call.limit != 100 {
		t.Fatalf("expected token limit 100, got %d", call.limit)
	}
}

func TestIPUsesIPLimit(t *testing.T) {
	store := &fakeStore{allowed: true}

	l := New(store, LimitConfig{
		IPLimit:           10,
		DefaultTokenLimit: 20,
		TokenLimits:       map[string]int{},
		Window:            time.Second,
		BlockDuration:     5 * time.Minute,
	})

	_, err := l.Allow(context.Background(), Identity{
		Type:  IdentityIP,
		Value: "127.0.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if store.calls[0].key != "ip:127.0.0.1" {
		t.Fatalf("unexpected key: %s", store.calls[0].key)
	}
	if store.calls[0].limit != 10 {
		t.Fatalf("expected IP limit 10, got %d", store.calls[0].limit)
	}
}
