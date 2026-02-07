package middleware

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// UserContextKey is the context key for storing the authenticated user
const UserContextKey contextKey = "user"

// Auth middleware handles authentication based on deployment phase
type Auth struct {
	initialized bool
	dockerMode  bool
	setupToken  string
}

// NewAuth creates a new auth middleware
func NewAuth(initialized, dockerMode bool, setupToken string) *Auth {
	return &Auth{
		initialized: initialized,
		dockerMode:  dockerMode,
		setupToken:  setupToken,
	}
}

// Middleware applies authentication logic
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Health check endpoint bypasses auth
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		// Static files bypass auth
		if len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/static/" {
			next.ServeHTTP(w, r)
			return
		}

		if !a.initialized {
			// Pre-init: Require setup token
			if !a.validateSetupToken(w, r) {
				return
			}
		} else if a.dockerMode {
			// Post-init Docker: Trust Authelia Remote-User header only from
			// private/Docker network IPs to prevent spoofing via direct access.
			if !isPrivateIP(r.RemoteAddr) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			username := r.Header.Get("Remote-User")
			if username == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			// Add user to context
			ctx := context.WithValue(r.Context(), UserContextKey, username)
			r = r.WithContext(ctx)
		}
		// Post-init standalone: Dev mode, no auth (warning logged elsewhere)

		next.ServeHTTP(w, r)
	})
}

// validateSetupToken validates the setup token from query param or cookie
func (a *Auth) validateSetupToken(w http.ResponseWriter, r *http.Request) bool {
	// Check query parameter first
	token := r.URL.Query().Get("token")
	if token == "" {
		// Check cookie
		cookie, err := r.Cookie("setup_token")
		if err == nil {
			token = cookie.Value
		}
	}

	// Validate token using constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(token), []byte(a.setupToken)) != 1 {
		http.Error(w, "Invalid or missing setup token", http.StatusUnauthorized)
		return false
	}

	// Set cookie if not already set (from query param)
	if token == r.URL.Query().Get("token") {
		http.SetCookie(w, &http.Cookie{
			Name:     "setup_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   3600, // 1 hour
		})
	}

	return true
}

// isPrivateIP checks whether a request originates from a private/Docker network address.
// This is used to ensure the Remote-User header is only trusted from the reverse proxy,
// not from direct public access.
func isPrivateIP(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	// Loopback (127.0.0.0/8, ::1)
	if ip.IsLoopback() {
		return true
	}

	// RFC 1918 and Docker default networks
	privateRanges := []struct {
		network *net.IPNet
	}{
		{mustParseCIDR("10.0.0.0/8")},
		{mustParseCIDR("172.16.0.0/12")},
		{mustParseCIDR("192.168.0.0/16")},
		{mustParseCIDR("fc00::/7")}, // IPv6 unique local
	}

	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}

	return false
}

func mustParseCIDR(s string) *net.IPNet {
	_, network, err := net.ParseCIDR(s)
	if err != nil {
		panic("invalid CIDR: " + s)
	}
	return network
}
