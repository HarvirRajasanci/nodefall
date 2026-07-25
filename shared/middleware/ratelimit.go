package middleware

import (
	"net"
	"net/http"
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter tracks a separate token-bucket limiter per client IP,
// so one abusive client can't exhaust the limit for everyone else.
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rate.Limiter
	r        rate.Limit
	burst    int
}

// NewRateLimiter creates a limiter allowing requestsPerSecond sustained
// requests per IP, with burst allowed to spike briefly above that.
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		r:        rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}

// Wrap rejects requests over the limit with 429 Too Many Requests,
// tracked separately per client IP.
func (rl *RateLimiter) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.getLimiter(clientIP(r)).Allow() {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	l, ok := rl.limiters[key]
	if !ok {
		l = rate.NewLimiter(rl.r, rl.burst)
		rl.limiters[key] = l
	}
	return l
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
