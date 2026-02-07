package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/time/rate"
)

// TestAuthPreInitValidToken verifies pre-init auth with valid token
func TestAuthPreInitValidToken(t *testing.T) {
	auth := NewAuth(false, false, "test-token-123")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Test with query parameter
	req := httptest.NewRequest(http.MethodGet, "/?token=test-token-123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that cookie was set
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "setup_token" && c.Value == "test-token-123" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected setup_token cookie to be set")
	}
}

// TestAuthPreInitInvalidToken verifies pre-init auth rejects invalid token
func TestAuthPreInitInvalidToken(t *testing.T) {
	auth := NewAuth(false, false, "test-token-123")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/?token=wrong-token", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestAuthPreInitMissingToken verifies pre-init auth rejects missing token
func TestAuthPreInitMissingToken(t *testing.T) {
	auth := NewAuth(false, false, "test-token-123")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestAuthPreInitCookieToken verifies pre-init auth with cookie token
func TestAuthPreInitCookieToken(t *testing.T) {
	auth := NewAuth(false, false, "test-token-123")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "setup_token", Value: "test-token-123"})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestAuthHealthBypass verifies health endpoint bypasses auth
func TestAuthHealthBypass(t *testing.T) {
	auth := NewAuth(false, false, "test-token-123")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("healthy"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestAuthStaticBypass verifies static files bypass auth
func TestAuthStaticBypass(t *testing.T) {
	auth := NewAuth(false, false, "test-token-123")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/static/css/main.css", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestAuthPostInitDockerMode verifies post-init Docker mode auth
func TestAuthPostInitDockerMode(t *testing.T) {
	auth := NewAuth(true, true, "")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(UserContextKey)
		if user != "testuser" {
			t.Errorf("expected user 'testuser' in context, got %v", user)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.17.0.2:12345" // Docker network IP
	req.Header.Set("Remote-User", "testuser")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestAuthPostInitDockerModeNoHeader verifies Docker mode rejects missing header from private IP
func TestAuthPostInitDockerModeNoHeader(t *testing.T) {
	auth := NewAuth(true, true, "")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.17.0.2:12345" // Private IP but no Remote-User header
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestAuthPostInitDevMode verifies post-init dev mode allows all
func TestAuthPostInitDevMode(t *testing.T) {
	auth := NewAuth(true, false, "")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestLoggingMiddleware verifies logging middleware captures request info
func TestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	logOutput := buf.String()

	if !strings.Contains(logOutput, "GET") {
		t.Error("log should contain request method")
	}

	if !strings.Contains(logOutput, "/test/path") {
		t.Error("log should contain request path")
	}

	if !strings.Contains(logOutput, "200") {
		t.Error("log should contain status code")
	}
}

// TestLoggingMiddlewareStatusCode verifies logging captures different status codes
func TestLoggingMiddlewareStatusCode(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	logOutput := buf.String()

	if !strings.Contains(logOutput, "404") {
		t.Error("log should contain 404 status code")
	}
}

// TestResponseWriterWrapper verifies the response writer wrapper
func TestResponseWriterWrapper(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rw.statusCode)
	}

	// Test Write
	n, err := rw.Write([]byte("test data"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 9 {
		t.Errorf("expected 9 bytes written, got %d", n)
	}
	if rw.written != 9 {
		t.Errorf("expected written=9, got %d", rw.written)
	}
}

// TestRecoveryMiddleware verifies panic recovery
func TestRecoveryMiddleware(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	// Should not panic
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "Panic recovered") {
		t.Error("log should contain panic recovery message")
	}

	if !strings.Contains(logOutput, "test panic") {
		t.Error("log should contain panic message")
	}
}

// TestRecoveryMiddlewareNoPanic verifies normal requests pass through
func TestRecoveryMiddlewareNoPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("expected body 'OK', got %q", w.Body.String())
	}
}

// TestAuthPreInitPartialTokenRejected verifies that a token sharing a prefix is still rejected
func TestAuthPreInitPartialTokenRejected(t *testing.T) {
	auth := NewAuth(false, false, "test-token-123")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Try a token that shares a prefix
	req := httptest.NewRequest(http.MethodGet, "/?token=test-token-124", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for partial match, got %d", w.Code)
	}
}

// TestAuthPreInitEmptyTokenRejected verifies empty token against non-empty setup token is rejected
func TestAuthPreInitEmptyTokenRejected(t *testing.T) {
	auth := NewAuth(false, false, "test-token-123")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Empty token via query param
	req := httptest.NewRequest(http.MethodGet, "/?token=", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401 for empty token, got %d", w.Code)
	}
}

// TestNewAuth verifies auth constructor
func TestNewAuth(t *testing.T) {
	auth := NewAuth(true, true, "token123")

	if !auth.initialized {
		t.Error("expected initialized=true")
	}

	if !auth.dockerMode {
		t.Error("expected dockerMode=true")
	}

	if auth.setupToken != "token123" {
		t.Errorf("expected setupToken='token123', got %q", auth.setupToken)
	}
}

// TestIsPrivateIP verifies private IP detection
func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		addr     string
		expected bool
	}{
		{"127.0.0.1:8080", true},
		{"10.0.0.1:8080", true},
		{"172.17.0.2:8080", true},      // Docker default bridge
		{"172.20.0.5:8080", true},       // Docker custom network
		{"192.168.1.100:8080", true},
		{"8.8.8.8:8080", false},         // Google DNS - public
		{"1.1.1.1:8080", false},         // Cloudflare DNS - public
		{"203.0.113.1:8080", false},     // TEST-NET - public
		{"[::1]:8080", true},            // IPv6 loopback
		{"[fd00::1]:8080", true},        // IPv6 unique local
		{"[2001:db8::1]:8080", false},   // IPv6 documentation - public
		{"invalid", false},              // Unparseable
	}

	for _, tt := range tests {
		result := isPrivateIP(tt.addr)
		if result != tt.expected {
			t.Errorf("isPrivateIP(%q) = %v, expected %v", tt.addr, result, tt.expected)
		}
	}
}

// TestAuthDockerModeRejectsPublicIP verifies Docker mode rejects non-private IPs
func TestAuthDockerModeRejectsPublicIP(t *testing.T) {
	auth := NewAuth(true, true, "")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "8.8.8.8:12345"
	req.Header.Set("Remote-User", "attacker")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for public IP, got %d", w.Code)
	}
}

// TestAuthDockerModeAllowsPrivateIP verifies Docker mode accepts private IPs with header
func TestAuthDockerModeAllowsPrivateIP(t *testing.T) {
	auth := NewAuth(true, true, "")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "172.17.0.2:12345"
	req.Header.Set("Remote-User", "admin")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for private IP with header, got %d", w.Code)
	}
}

// TestAuthDockerModeLoopback verifies Docker mode accepts loopback
func TestAuthDockerModeLoopback(t *testing.T) {
	auth := NewAuth(true, true, "")

	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("Remote-User", "admin")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for loopback, got %d", w.Code)
	}
}

// TestRateLimiterAllowsNormalTraffic verifies requests within limits pass through
func TestRateLimiterAllowsNormalTraffic(t *testing.T) {
	rl := NewRateLimiter(10, 20)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// First request should pass
	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestRateLimiterBlocksExcessiveTraffic verifies requests over limit are rejected
func TestRateLimiterBlocksExcessiveTraffic(t *testing.T) {
	// Very restrictive: 1 req/sec, burst of 2
	rl := NewRateLimiter(1, 2)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	blocked := 0
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code == http.StatusTooManyRequests {
			blocked++
		}
	}

	if blocked == 0 {
		t.Error("rate limiter should have blocked some requests")
	}
}

// TestRateLimiterPerIP verifies different IPs have separate limits
func TestRateLimiterPerIP(t *testing.T) {
	// Very restrictive: 1 req/sec, burst of 1
	rl := NewRateLimiter(1, 1)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust limit for IP1
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// IP2 should still be allowed
	req := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	req.RemoteAddr = "10.0.0.2:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("different IP should not be rate limited, got status %d", w.Code)
	}
}

// TestRateLimiterHealthBypass verifies health endpoint bypasses rate limiting
func TestRateLimiterHealthBypass(t *testing.T) {
	// Very restrictive
	rl := NewRateLimiter(1, 1)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/some-path", nil)
		req.RemoteAddr = "10.0.0.3:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Health check should still pass
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.RemoteAddr = "10.0.0.3:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health endpoint should bypass rate limiting, got status %d", w.Code)
	}
}

// TestRateLimiterStaticBypass verifies static assets bypass rate limiting
func TestRateLimiterStaticBypass(t *testing.T) {
	rl := NewRateLimiter(1, 1)

	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
		req.RemoteAddr = "10.0.0.4:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}

	// Static file should still pass
	req := httptest.NewRequest(http.MethodGet, "/static/css/main.css", nil)
	req.RemoteAddr = "10.0.0.4:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("static endpoint should bypass rate limiting, got status %d", w.Code)
	}
}

// TestNewRateLimiter verifies constructor
func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(5), 10)

	if rl == nil {
		t.Fatal("NewRateLimiter should return non-nil")
	}

	if rl.rate != rate.Limit(5) {
		t.Errorf("expected rate 5, got %v", rl.rate)
	}

	if rl.burst != 10 {
		t.Errorf("expected burst 10, got %d", rl.burst)
	}
}

// TestCSRFAllowsGETWithoutToken verifies GET requests pass without CSRF token
func TestCSRFAllowsGETWithoutToken(t *testing.T) {
	csrf := NewCSRF()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET without CSRF token should pass, got %d", w.Code)
	}

	// Should set a CSRF cookie
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "csrf_token" && c.Value != "" {
			found = true
			break
		}
	}
	if !found {
		t.Error("GET response should set csrf_token cookie")
	}
}

// TestCSRFBlocksPOSTWithoutToken verifies POST requests are blocked without CSRF token
func TestCSRFBlocksPOSTWithoutToken(t *testing.T) {
	csrf := NewCSRF()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/config/save", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("POST without CSRF token should be 403, got %d", w.Code)
	}
}

// TestCSRFBlocksPOSTWithMismatchedToken verifies mismatched tokens are rejected
func TestCSRFBlocksPOSTWithMismatchedToken(t *testing.T) {
	csrf := NewCSRF()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/config/save", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "token-a"})
	req.Header.Set("X-CSRF-Token", "token-b")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("POST with mismatched CSRF tokens should be 403, got %d", w.Code)
	}
}

// TestCSRFAllowsPOSTWithMatchingToken verifies matching tokens pass
func TestCSRFAllowsPOSTWithMatchingToken(t *testing.T) {
	csrf := NewCSRF()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token := "my-csrf-token-value"
	req := httptest.NewRequest(http.MethodPost, "/api/config/save", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	req.Header.Set("X-CSRF-Token", token)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST with matching CSRF token should be 200, got %d", w.Code)
	}
}

// TestCSRFAllowsHealthEndpoint verifies health bypasses CSRF
func TestCSRFAllowsHealthEndpoint(t *testing.T) {
	csrf := NewCSRF()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health endpoint POST should bypass CSRF, got %d", w.Code)
	}
}

// TestCSRFBlocksDELETEWithoutToken verifies DELETE is also protected
func TestCSRFBlocksDELETEWithoutToken(t *testing.T) {
	csrf := NewCSRF()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodDelete, "/api/backup/delete/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("DELETE without CSRF token should be 403, got %d", w.Code)
	}
}

// TestCSRFFormValueFallback verifies CSRF token can come from form field
func TestCSRFFormValueFallback(t *testing.T) {
	csrf := NewCSRF()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token := "form-csrf-token"
	form := strings.NewReader("csrf_token=" + token + "&name=test")
	req := httptest.NewRequest(http.MethodPost, "/setup/domain", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("POST with CSRF token in form field should be 200, got %d", w.Code)
	}
}

// TestCSRFCookieNotHttpOnly verifies CSRF cookie is readable by JavaScript
func TestCSRFCookieNotHttpOnly(t *testing.T) {
	csrf := NewCSRF()

	handler := csrf.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "csrf_token" {
			if c.HttpOnly {
				t.Error("CSRF cookie must NOT be HttpOnly (must be readable by JavaScript)")
			}
			if c.SameSite != http.SameSiteStrictMode {
				t.Error("CSRF cookie should have SameSite=Strict")
			}
			return
		}
	}
	t.Error("csrf_token cookie not found")
}

// TestExtractIP verifies IP extraction from requests
func TestSecurityHeadersCSP(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header should be set")
	}
	if !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("CSP should contain default-src 'self', got: %s", csp)
	}
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("CSP should contain frame-ancestors 'none', got: %s", csp)
	}
}

func TestSecurityHeadersPresent(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":       "DENY",
		"Referrer-Policy":       "strict-origin-when-cross-origin",
	}

	for header, expected := range expectedHeaders {
		actual := w.Header().Get(header)
		if actual != expected {
			t.Errorf("%s: expected %q, got %q", header, expected, actual)
		}
	}

	// Permissions-Policy should be present
	pp := w.Header().Get("Permissions-Policy")
	if pp == "" {
		t.Error("Permissions-Policy header should be set")
	}
}

func TestSecurityHeadersDoNotBreakHandler(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "test")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Errorf("expected body 'hello', got %q", w.Body.String())
	}
	if w.Header().Get("X-Custom") != "test" {
		t.Error("handler's custom header should still be present")
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		remoteAddr string
		expected   string
	}{
		{"192.168.1.1:12345", "192.168.1.1"},
		{"10.0.0.1:80", "10.0.0.1"},
		{"[::1]:8080", "::1"},
		{"invalid-no-port", "invalid-no-port"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = tt.remoteAddr
		ip := extractIP(req)
		if ip != tt.expected {
			t.Errorf("extractIP(%q) = %q, expected %q", tt.remoteAddr, ip, tt.expected)
		}
	}
}
