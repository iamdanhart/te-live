package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecureHeaders_SetsAllHeaders(t *testing.T) {
	h := SecureHeaders(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, "DENY", rr.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", rr.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "strict-origin", rr.Header().Get("Referrer-Policy"))
	assert.Contains(t, rr.Header().Get("Content-Security-Policy"), "default-src 'self'")
	assert.Contains(t, rr.Header().Get("Content-Security-Policy"), "script-src 'self'")
	assert.Contains(t, rr.Header().Get("Content-Security-Policy"), "style-src 'self' https://fonts.googleapis.com")
}

func TestSecureHeaders_CallsNextHandler(t *testing.T) {
	h := SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTeapot, rr.Code)
}