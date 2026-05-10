package middleware

import (
	"log/slog"
	"net/http"
	"net/url"
)

// RequireSameOrigin blocks POST requests whose Origin or Referer header does not
// match an allowed host. When allowedHosts is empty, all requests are allowed
// (dev mode). Requests with neither header are allowed through.
func RequireSameOrigin(allowedHosts []string) func(http.Handler) http.Handler {
	if len(allowedHosts) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	allowed := make(map[string]bool, len(allowedHosts))
	for _, h := range allowedHosts {
		allowed[h] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Origin")
			if header == "" {
				header = r.Header.Get("Referer")
			}
			if header != "" {
				u, err := url.Parse(header)
				if err != nil || !allowed[u.Host] {
					slog.Warn("CSRF check failed", "ip", r.RemoteAddr, "method", r.Method, "path", r.URL.Path, "header", header)
					http.Error(w, "forbidden", http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}