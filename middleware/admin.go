package middleware

import (
	"net/http"
	"os"
)

// AdminAuth is middleware that requires the request to present the correct
// password via HTTP Basic Auth. The expected password is read from the
// ADMIN_PASSWORD environment variable. If unset, the admin routes are
// inaccessible.
func AdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pass, ok := r.BasicAuth()
		if !ok || pass != os.Getenv("ADMIN_PASSWORD") {
			w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}