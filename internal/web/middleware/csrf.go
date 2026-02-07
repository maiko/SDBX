package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

const (
	csrfCookieName = "csrf_token"
	csrfHeaderName = "X-CSRF-Token"
	csrfTokenBytes = 32
)

// CSRF provides double-submit cookie CSRF protection.
// On GET requests, it sets a csrf_token cookie.
// On state-changing requests (POST, PUT, DELETE, PATCH), it validates that the
// X-CSRF-Token header matches the cookie value.
type CSRF struct{}

// NewCSRF creates a new CSRF middleware.
func NewCSRF() *CSRF {
	return &CSRF{}
}

// Middleware applies CSRF validation to state-changing requests.
func (c *CSRF) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF for safe methods and non-web paths
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			c.ensureToken(w, r)
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF for health endpoint
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Validate CSRF token on state-changing methods
		cookieToken, err := r.Cookie(csrfCookieName)
		if err != nil || cookieToken.Value == "" {
			http.Error(w, "CSRF token missing", http.StatusForbidden)
			return
		}

		headerToken := r.Header.Get(csrfHeaderName)
		if headerToken == "" {
			// Also check form value for non-AJAX form submissions
			headerToken = r.FormValue("csrf_token")
		}

		if headerToken == "" {
			http.Error(w, "CSRF token missing", http.StatusForbidden)
			return
		}

		if subtle.ConstantTimeCompare([]byte(cookieToken.Value), []byte(headerToken)) != 1 {
			http.Error(w, "CSRF token mismatch", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ensureToken sets a CSRF token cookie if one doesn't exist.
func (c *CSRF) ensureToken(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie(csrfCookieName); err == nil {
		return // Cookie already exists
	}

	token, err := generateCSRFToken()
	if err != nil {
		return // Fail open for token generation (GET requests are safe)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Must be readable by JavaScript to send in header
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   3600,
	})
}

func generateCSRFToken() (string, error) {
	b := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
