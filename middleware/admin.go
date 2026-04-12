package middleware

import (
	"net/http"
	"os"
)

// AdminAuth returns middleware that requires HTTP Basic Auth. When enforce is
// false, all requests are allowed through without checking credentials.
func AdminAuth(enforce bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if enforce {
			_, pass, ok := r.BasicAuth()
			if !ok || pass != os.Getenv("ADMIN_PASSWORD") {
				w.Header().Set("WWW-Authenticate", `Basic realm="sign in"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
