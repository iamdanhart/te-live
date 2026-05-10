package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// FailureLimiter blocks an IP after too many unauthorized responses within a window.
// Unlike RateLimiter, successful requests are never penalized.
type FailureLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time
	window   time.Duration
	maxFails int
}

func NewFailureLimiter(ctx context.Context, window time.Duration, maxFails int) *FailureLimiter {
	fl := &FailureLimiter{
		failures: make(map[string][]time.Time),
		window:   window,
		maxFails: maxFails,
	}
	go fl.cleanup(ctx)
	return fl
}

func (fl *FailureLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)

		fl.mu.Lock()
		var recent []time.Time
		for _, t := range fl.failures[ip] {
			if time.Since(t) < fl.window {
				recent = append(recent, t)
			}
		}
		fl.failures[ip] = recent

		if len(recent) >= fl.maxFails {
			fl.mu.Unlock()
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		fl.mu.Unlock()

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		if rec.status == http.StatusUnauthorized {
			fl.mu.Lock()
			fl.failures[ip] = append(fl.failures[ip], time.Now())
			fl.mu.Unlock()
		}
	})
}

func (fl *FailureLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fl.mu.Lock()
			for ip, times := range fl.failures {
				var recent []time.Time
				for _, t := range times {
					if time.Since(t) < fl.window {
						recent = append(recent, t)
					}
				}
				if len(recent) == 0 {
					delete(fl.failures, ip)
				} else {
					fl.failures[ip] = recent
				}
			}
			fl.mu.Unlock()
		}
	}
}