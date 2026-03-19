package middleware

import "net/http"

// SecurityHeaders adds standard security headers to all responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		// Content-Security-Policy: restrict resource loading to same origin,
		// allow inline styles (needed for Go templates), and the htmx CDN.
		h.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws: wss:; frame-ancestors 'none'")

		// Prevent MIME type sniffing
		h.Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		h.Set("X-Frame-Options", "DENY")

		// Control referrer information
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Opt out of browser features not needed
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		next.ServeHTTP(w, r)
	})
}
