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

// makeRequest sends a request through the limiter, carrying any session cookie
// set on a previous response.
func makeRequest(handler http.Handler, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// sessionCookie extracts the session cookie from a response, if present.
func sessionCookie(rr *httptest.ResponseRecorder) *http.Cookie {
	resp := rr.Result()
	for _, c := range resp.Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	return nil
}

// RateLimiter is constructed directly rather than via NewRateLimiter to avoid
// spinning up the background cleanup goroutine in tests.

func TestLimit_FirstRequestAllowed(t *testing.T) {
	rl := &RateLimiter{last: make(map[string]time.Time), cooldown: time.Minute, enforce: true}
	rr := makeRequest(rl.Limit(okHandler()), nil)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestLimit_SecondRequestWithinCooldownRejected(t *testing.T) {
	rl := &RateLimiter{last: make(map[string]time.Time), cooldown: time.Minute, enforce: true}
	handler := rl.Limit(okHandler())

	first := makeRequest(handler, nil)
	assert.Equal(t, http.StatusOK, first.Code)

	second := makeRequest(handler, sessionCookie(first))
	assert.Equal(t, http.StatusTooManyRequests, second.Code)
}

func TestLimit_RequestAfterCooldownAllowed(t *testing.T) {
	rl := &RateLimiter{last: make(map[string]time.Time), cooldown: time.Millisecond, enforce: true}
	handler := rl.Limit(okHandler())

	first := makeRequest(handler, nil)
	assert.Equal(t, http.StatusOK, first.Code)

	time.Sleep(5 * time.Millisecond)

	second := makeRequest(handler, sessionCookie(first))
	assert.Equal(t, http.StatusOK, second.Code)
}

func TestLimit_NotEnforcedAlwaysAllows(t *testing.T) {
	rl := &RateLimiter{last: make(map[string]time.Time), cooldown: time.Minute, enforce: false}
	handler := rl.Limit(okHandler())

	first := makeRequest(handler, nil)
	assert.Equal(t, http.StatusOK, first.Code)

	second := makeRequest(handler, sessionCookie(first))
	assert.Equal(t, http.StatusOK, second.Code)
}