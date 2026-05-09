package middleware

import "net/http"

// AdminAuth returns middleware that requires HTTP Basic Auth. When enforce is
// false, all requests are allowed through without checking credentials.
// check is called with the submitted password; it should return true if valid.
func AdminAuth(enforce bool, check func(string) bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if enforce {
			_, pass, ok := r.BasicAuth()
			if !ok || !check(pass) {
				w.Header().Set("WWW-Authenticate", `Basic realm="sign in"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}