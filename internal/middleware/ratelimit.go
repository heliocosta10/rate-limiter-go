package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/heliocosta10/rate-limiter-go/internal/limiter"
)

const blockedMessage = "you have reached the maximum number of requests or actions allowed within a certain time frame"

type RateLimitMiddleware struct {
	limiter *limiter.RateLimiter
}

func NewRateLimitMiddleware(l *limiter.RateLimiter) *RateLimitMiddleware {
	return &RateLimitMiddleware{limiter: l}
}

func (m *RateLimitMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := resolveIdentity(r)

		allowed, err := m.limiter.Allow(r.Context(), identity)
		if err != nil {
			http.Error(w, "rate limiter internal error", http.StatusInternalServerError)
			return
		}

		if !allowed {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(blockedMessage))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func resolveIdentity(r *http.Request) limiter.Identity {
	if token := strings.TrimSpace(r.Header.Get("API_KEY")); token != "" {
		return limiter.Identity{
			Type:  limiter.IdentityToken,
			Value: token,
		}
	}

	return limiter.Identity{
		Type:  limiter.IdentityIP,
		Value: clientIP(r),
	}
}

func clientIP(r *http.Request) string {
	// X-Forwarded-For is intentionally supported for reverse-proxy deployments.
	// In a trusted-proxy environment this should be sanitized by the proxy.
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		if first := strings.TrimSpace(strings.Split(forwarded, ",")[0]); first != "" {
			return first
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
