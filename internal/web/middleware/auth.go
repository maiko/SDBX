package middleware

import (
	"context"
	"crypto/subtle"
	"net"
	"net/http"
)

const (
	// setupTokenCookieMaxAge is how long the setup token cookie lasts (1 hour).
	setupTokenCookieMaxAge = 3600
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

// validateSetupToken validates the setup token from query param or cookie.
// Returns true if the request should proceed, false if the response has been
// written (either a redirect or an error).
func (a *Auth) validateSetupToken(w http.ResponseWriter, r *http.Request) bool {
	// Check query parameter first
	queryToken := r.URL.Query().Get("token")
	if queryToken != "" {
		// Validate query token
		if subtle.ConstantTimeCompare([]byte(queryToken), []byte(a.setupToken)) != 1 {
			http.Error(w, "Invalid or missing setup token", http.StatusUnauthorized)
			return false
		}

		// Set cookie and redirect to strip token from URL (prevents exposure
		// in browser history, bookmarks, referrer headers, and server logs)
		http.SetCookie(w, &http.Cookie{
			Name:     "setup_token",
			Value:    queryToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   isHTTPS(r),
			SameSite: http.SameSiteStrictMode,
			MaxAge:   setupTokenCookieMaxAge,
		})

		// Build redirect URL without the token parameter
		cleanURL := *r.URL
		q := cleanURL.Query()
		q.Del("token")
		cleanURL.RawQuery = q.Encode()
		http.Redirect(w, r, cleanURL.String(), http.StatusFound)
		return false // Response written (redirect)
	}

	// Check cookie
	cookie, err := r.Cookie("setup_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Invalid or missing setup token", http.StatusUnauthorized)
		return false
	}

	// Validate cookie token
	if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(a.setupToken)) != 1 {
		http.Error(w, "Invalid or missing setup token", http.StatusUnauthorized)
		return false
	}

	return true
}

// isHTTPS returns true if the request was made over HTTPS, either directly
// (r.TLS != nil) or via a reverse proxy (X-Forwarded-Proto header).
func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}

// privateNetworks holds parsed CIDR ranges for private/Docker network detection.
// Parsed once at init to avoid repeated allocation on every request.
var privateNetworks []*net.IPNet

func init() {
	cidrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"fc00::/7", // IPv6 unique local
	}
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic("invalid CIDR: " + cidr)
		}
		privateNetworks = append(privateNetworks, network)
	}
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

	for _, network := range privateNetworks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}
