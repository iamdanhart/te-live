package middleware

import "net/http"

// SecureHeaders sets defensive HTTP response headers on every request.
//
// X-Frame-Options: DENY — prevents the app from being embedded in an iframe,
// blocking clickjacking attacks where an attacker tricks a user into
// interacting with a hidden frame.
//
// X-Content-Type-Options: nosniff — prevents browsers from guessing the MIME
// type of a response and executing it as something other than what was declared.
//
// Referrer-Policy: strict-origin — sends only the origin (no path) in the
// Referer header on cross-origin requests. This is intentionally permissive
// enough to keep the header present for CSRF checks on requests from
// cross-origin allowed hosts (e.g. the brewery's domain), while avoiding
// leaking full URL paths. "strict" means the header is omitted entirely on
// HTTPS→HTTP downgrades, which cannot happen here as Fly.io enforces HTTPS.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' https://fonts.googleapis.com; "+
				"font-src https://fonts.gstatic.com; "+
				"img-src 'self' data:; "+
				"connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}