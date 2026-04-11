package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

// rateLimiter tracks the last request time per session cookie and rejects
// requests that arrive within the cooldown window.
type rateLimiter struct {
	mu       sync.Mutex
	last     map[string]time.Time
	cooldown time.Duration
}

// newRateLimiter creates a rateLimiter with the given cooldown and starts
// a background goroutine to periodically evict expired entries.
func newRateLimiter(cooldown time.Duration) *rateLimiter {
	rl := &rateLimiter{
		last:     make(map[string]time.Time),
		cooldown: cooldown,
	}
	go rl.cleanup()
	return rl
}

// sessionID returns the value of the session cookie, creating and setting
// one on the response if the request does not already have one.
func sessionID(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie("session"); err == nil {
		return cookie.Value
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// should never happen
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return ""
	}
	id := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    id,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	return id
}

// limit is middleware that wraps the given handler. It allows the request
// through if the session has not made a request within the cooldown window,
// otherwise it responds with 429 Too Many Requests.
func (rl *rateLimiter) limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := sessionID(w, r)

		rl.mu.Lock()
		if last, ok := rl.last[id]; ok && time.Since(last) < rl.cooldown {
			rl.mu.Unlock()
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		rl.last[id] = time.Now()
		rl.mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// cleanup runs forever, removing sessions from the map whose last request was
// longer ago than the cooldown. This prevents the map from growing unbounded.
func (rl *rateLimiter) cleanup() {
	for range time.Tick(time.Minute) {
		rl.mu.Lock()
		for id, last := range rl.last {
			if time.Since(last) > rl.cooldown {
				delete(rl.last, id)
			}
		}
		rl.mu.Unlock()
	}
}
