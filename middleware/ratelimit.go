package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

// Limiter is implemented by any middleware that can wrap an http.Handler
// to enforce request limits.
type Limiter interface {
	Limit(next http.Handler) http.Handler
}

// RateLimiter tracks the last request time per client IP and rejects
// requests that arrive within the cooldown window.
type RateLimiter struct {
	mu       sync.Mutex
	last     map[string]time.Time
	cooldown time.Duration
	enforce  bool
}

// NewRateLimiter creates a RateLimiter with the given cooldown and starts
// a background goroutine to periodically evict expired entries. When enforce
// is false, request times are still tracked but the limit is never applied.
func NewRateLimiter(ctx context.Context, cooldown time.Duration, enforce bool) *RateLimiter {
	rl := &RateLimiter{
		last:     make(map[string]time.Time),
		cooldown: cooldown,
		enforce:  enforce,
	}
	go rl.cleanup(ctx)
	return rl
}

// clientIP returns the real client IP. Fly-Client-IP is preferred over RemoteAddr
// because RemoteAddr reflects the proxy, and Fly strips any client-supplied value of this header.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("Fly-Client-IP"); ip != "" {
		return ip
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// Limit is middleware that wraps the given handler. It allows the request
// through if the client IP has not made a request within the cooldown window,
// otherwise it responds with 429 Too Many Requests.
func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		rl.mu.Lock()
		if rl.enforce {
			if last, ok := rl.last[ip]; ok && time.Since(last) < rl.cooldown {
				rl.mu.Unlock()
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
		}
		rl.last[ip] = time.Now()
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// cleanup runs forever, removing IPs from the map whose last request was
// longer ago than the cooldown. This prevents the map from growing unbounded.
func (rl *RateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			for ip, last := range rl.last {
				if time.Since(last) > rl.cooldown {
					delete(rl.last, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}