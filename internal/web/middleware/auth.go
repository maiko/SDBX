package middleware

import (
	"context"
	"net/http"
)

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
			// Post-init Docker: Trust Authelia Remote-User header
			username := r.Header.Get("Remote-User")
			if username == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			// Add user to context
			ctx := context.WithValue(r.Context(), "user", username)
			r = r.WithContext(ctx)
		} else {
			// Post-init standalone: Dev mode, no auth but log warning
			// In production, this path should not be used
		}

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

	// Validate token
	if token != a.setupToken {
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
