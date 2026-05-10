package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeAdminRequest(handler http.Handler, user, pass string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func TestAdminAuth_NotEnforcedAlwaysAllows(t *testing.T) {
	handler := AdminAuth(false, func(context.Context, string) bool { return false }, okHandler())
	rr := makeAdminRequest(handler, "", "")
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAdminAuth_BlocksWithNoCredentials(t *testing.T) {
	handler := AdminAuth(true, func(context.Context, string) bool { return true }, okHandler())
	rr := makeAdminRequest(handler, "", "")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, `Basic realm="sign in"`, rr.Header().Get("WWW-Authenticate"))
}

func TestAdminAuth_BlocksWithWrongPassword(t *testing.T) {
	handler := AdminAuth(true, func(_ context.Context, pass string) bool { return pass == "correct" }, okHandler())
	rr := makeAdminRequest(handler, "user", "wrong")
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAdminAuth_AllowsWithCorrectPassword(t *testing.T) {
	handler := AdminAuth(true, func(_ context.Context, pass string) bool { return pass == "correct" }, okHandler())
	rr := makeAdminRequest(handler, "user", "correct")
	assert.Equal(t, http.StatusOK, rr.Code)
}