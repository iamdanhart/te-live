package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequireSameOrigin_EmptyAllowedHosts_AllowsAll(t *testing.T) {
	h := RequireSameOrigin(nil)(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireSameOrigin_MatchingOrigin_Allows(t *testing.T) {
	h := RequireSameOrigin([]string{"example.com"})(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireSameOrigin_MismatchedOrigin_Blocks(t *testing.T) {
	h := RequireSameOrigin([]string{"example.com"})(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireSameOrigin_MatchingReferer_Allows(t *testing.T) {
	h := RequireSameOrigin([]string{"example.com"})(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Referer", "https://example.com/page")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireSameOrigin_MismatchedReferer_Blocks(t *testing.T) {
	h := RequireSameOrigin([]string{"example.com"})(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Referer", "https://evil.com/page")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireSameOrigin_OriginTakesPrecedenceOverReferer(t *testing.T) {
	h := RequireSameOrigin([]string{"example.com"})(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "https://evil.com")
	req.Header.Set("Referer", "https://example.com/page")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestRequireSameOrigin_NoHeaders_Allows(t *testing.T) {
	h := RequireSameOrigin([]string{"example.com"})(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireSameOrigin_UnparsableOrigin_Blocks(t *testing.T) {
	h := RequireSameOrigin([]string{"example.com"})(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Origin", "://not a url")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}