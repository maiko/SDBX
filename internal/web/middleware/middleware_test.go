package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
	req.Header.Set("Remote-User", "testuser")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

// TestAuthPostInitDockerModeNoHeader verifies Docker mode rejects missing header
func TestAuthPostInitDockerModeNoHeader(t *testing.T) {
	auth := NewAuth(true, true, "")

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
