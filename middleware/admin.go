package middleware

import (
	"context"
	"log/slog"
	"net/http"
)

// AdminAuth returns middleware that requires HTTP Basic Auth. When enforce is
// false, all requests are allowed through without checking credentials.
// check is called with the request context and submitted password; it should return true if valid.
func AdminAuth(enforce bool, check func(context.Context, string) bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if enforce {
			_, pass, ok := r.BasicAuth()
			if !ok || !check(r.Context(), pass) {
				slog.Warn("unauthorized host auth attempt", "ip", r.RemoteAddr, "method", r.Method, "path", r.URL.Path)
				w.Header().Set("WWW-Authenticate", `Basic realm="sign in"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}