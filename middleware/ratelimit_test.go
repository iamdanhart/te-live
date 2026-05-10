package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func makeRequest(handler http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.RemoteAddr = remoteAddr
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// RateLimiter is constructed directly rather than via NewRateLimiter to avoid
// spinning up the background cleanup goroutine in tests.

func TestLimit_FirstRequestAllowed(t *testing.T) {
	rl := &RateLimiter{last: make(map[string]time.Time), cooldown: time.Minute, enforce: true}
	rr := makeRequest(rl.Limit(okHandler()), "1.2.3.4:1000")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestLimit_SecondRequestWithinCooldownRejected(t *testing.T) {
	rl := &RateLimiter{last: make(map[string]time.Time), cooldown: time.Minute, enforce: true}
	handler := rl.Limit(okHandler())

	first := makeRequest(handler, "1.2.3.4:1000")
	assert.Equal(t, http.StatusOK, first.Code)

	second := makeRequest(handler, "1.2.3.4:1001")
	assert.Equal(t, http.StatusTooManyRequests, second.Code)
}

func TestLimit_RequestAfterCooldownAllowed(t *testing.T) {
	rl := &RateLimiter{last: make(map[string]time.Time), cooldown: time.Millisecond, enforce: true}
	handler := rl.Limit(okHandler())

	first := makeRequest(handler, "1.2.3.4:1000")
	assert.Equal(t, http.StatusOK, first.Code)

	time.Sleep(5 * time.Millisecond)

	second := makeRequest(handler, "1.2.3.4:1001")
	assert.Equal(t, http.StatusOK, second.Code)
}

func TestLimit_DifferentIPsAllowed(t *testing.T) {
	rl := &RateLimiter{last: make(map[string]time.Time), cooldown: time.Minute, enforce: true}
	handler := rl.Limit(okHandler())

	first := makeRequest(handler, "1.2.3.4:1000")
	assert.Equal(t, http.StatusOK, first.Code)

	second := makeRequest(handler, "5.6.7.8:1000")
	assert.Equal(t, http.StatusOK, second.Code)
}

func TestLimit_NotEnforcedAlwaysAllows(t *testing.T) {
	rl := &RateLimiter{last: make(map[string]time.Time), cooldown: time.Minute, enforce: false}
	handler := rl.Limit(okHandler())

	first := makeRequest(handler, "1.2.3.4:1000")
	assert.Equal(t, http.StatusOK, first.Code)

	second := makeRequest(handler, "1.2.3.4:1001")
	assert.Equal(t, http.StatusOK, second.Code)
}