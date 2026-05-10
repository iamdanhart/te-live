package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func unauthorizedHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

func makeFailureRequest(handler http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// FailureLimiter is constructed directly to avoid spinning up the background cleanup goroutine.

func TestFailureLimiter_AllowsBeforeThreshold(t *testing.T) {
	fl := &FailureLimiter{failures: make(map[string][]time.Time), window: time.Minute, maxFails: 3}
	handler := fl.Limit(unauthorizedHandler())

	for i := 0; i < 2; i++ {
		rr := makeFailureRequest(handler, "1.2.3.4:1000")
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	}
}

func TestFailureLimiter_BlocksAfterThreshold(t *testing.T) {
	fl := &FailureLimiter{failures: make(map[string][]time.Time), window: time.Minute, maxFails: 3}
	handler := fl.Limit(unauthorizedHandler())

	for i := 0; i < 3; i++ {
		makeFailureRequest(handler, "1.2.3.4:1000")
	}

	rr := makeFailureRequest(handler, "1.2.3.4:1000")
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestFailureLimiter_SuccessfulRequestsNotCounted(t *testing.T) {
	fl := &FailureLimiter{failures: make(map[string][]time.Time), window: time.Minute, maxFails: 3}
	handler := fl.Limit(okHandler())

	for i := 0; i < 10; i++ {
		rr := makeFailureRequest(handler, "1.2.3.4:1000")
		assert.Equal(t, http.StatusOK, rr.Code)
	}
}

func TestFailureLimiter_DifferentIPsIndependent(t *testing.T) {
	fl := &FailureLimiter{failures: make(map[string][]time.Time), window: time.Minute, maxFails: 3}
	handler := fl.Limit(unauthorizedHandler())

	for i := 0; i < 3; i++ {
		makeFailureRequest(handler, "1.2.3.4:1000")
	}

	rr := makeFailureRequest(handler, "5.6.7.8:1000")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestFailureLimiter_AllowsAfterWindowExpires(t *testing.T) {
	fl := &FailureLimiter{failures: make(map[string][]time.Time), window: 5 * time.Millisecond, maxFails: 3}
	handler := fl.Limit(unauthorizedHandler())

	for i := 0; i < 3; i++ {
		makeFailureRequest(handler, "1.2.3.4:1000")
	}

	time.Sleep(10 * time.Millisecond)

	rr := makeFailureRequest(handler, "1.2.3.4:1000")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}