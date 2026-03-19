package handlers

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestFormatServiceName verifies service name formatting
func TestFormatServiceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"radarr", "Radarr"},
		{"sonarr", "Sonarr"},
		{"qbittorrent", "Qbittorrent"},
		{"sdbx-webui", "Sdbx Webui"},
		{"my-cool-service", "My Cool Service"},
		{"", ""},
		{"a", "A"},
		{"ABC", "ABC"},
	}

	for _, tt := range tests {
		result := formatServiceName(tt.input)
		if result != tt.expected {
			t.Errorf("formatServiceName(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

// TestHttpError verifies error handling logs internally but returns generic message
func TestHttpError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	w := httptest.NewRecorder()
	err := &testError{msg: "sensitive database error"}

	httpError(w, "test-operation", err, http.StatusInternalServerError)

	// Check response doesn't contain sensitive info
	body := w.Body.String()
	if strings.Contains(body, "database") {
		t.Error("response should not contain sensitive error details")
	}

	if !strings.Contains(body, "internal error") {
		t.Error("response should contain generic error message")
	}

	// Check internal log contains the full error
	logOutput := buf.String()
	if !strings.Contains(logOutput, "sensitive database error") {
		t.Error("internal log should contain full error message")
	}

	if !strings.Contains(logOutput, "test-operation") {
		t.Error("internal log should contain operation context")
	}

	// Check status code
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// TestHttpErrorDifferentStatusCodes verifies different HTTP status codes
func TestHttpErrorDifferentStatusCodes(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	statusCodes := []int{
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, code := range statusCodes {
		w := httptest.NewRecorder()
		httpError(w, "test", &testError{msg: "error"}, code)

		if w.Code != code {
			t.Errorf("expected status %d, got %d", code, w.Code)
		}
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// TestJsonErrorHidesInternalDetails verifies jsonError returns generic message and logs internally
func TestJsonErrorHidesInternalDetails(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(nil)

	w := httptest.NewRecorder()
	jsonError(w, "Something went wrong", "test.context", &testError{msg: "secret internal: db connection failed at 10.0.0.1:5432"}, http.StatusInternalServerError)

	// Response should contain the user-facing message, not the internal error
	body := w.Body.String()
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
	if !strings.Contains(body, "Something went wrong") {
		t.Errorf("response should contain user message, got: %s", body)
	}
	if strings.Contains(body, "db connection failed") {
		t.Errorf("response must not contain internal error details, got: %s", body)
	}
	if strings.Contains(body, "10.0.0.1") {
		t.Errorf("response must not contain internal IP, got: %s", body)
	}

	// Log should contain the full error
	logOutput := buf.String()
	if !strings.Contains(logOutput, "db connection failed") {
		t.Errorf("log should contain internal error, got: %s", logOutput)
	}
	if !strings.Contains(logOutput, "test.context") {
		t.Errorf("log should contain context, got: %s", logOutput)
	}

	// Content-Type should be JSON
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", ct)
	}
}

// TestDashboardHandlerConstruction verifies dashboard handler can be created
func TestDashboardHandlerConstruction(t *testing.T) {
	handler := NewDashboardHandler(nil, nil, nil)

	if handler == nil {
		t.Error("NewDashboardHandler should return non-nil handler")
	}
}

// TestServicesHandlerConstruction verifies services handler can be created
func TestServicesHandlerConstruction(t *testing.T) {
	handler := NewServicesHandler(nil, nil, nil)

	if handler == nil {
		t.Error("NewServicesHandler should return non-nil handler")
	}
}

// TestLogsHandlerConstruction verifies logs handler can be created
func TestLogsHandlerConstruction(t *testing.T) {
	handler := NewLogsHandler(nil, nil, nil)

	if handler == nil {
		t.Error("NewLogsHandler should return non-nil handler")
	}
}

// TestAddonsHandlerConstruction verifies addons handler can be created
func TestAddonsHandlerConstruction(t *testing.T) {
	handler := NewAddonsHandler(nil, "", nil)

	if handler == nil {
		t.Error("NewAddonsHandler should return non-nil handler")
	}
}

// TestConfigHandlerConstruction verifies config handler can be created
func TestConfigHandlerConstruction(t *testing.T) {
	handler := NewConfigHandler("", nil)

	if handler == nil {
		t.Error("NewConfigHandler should return non-nil handler")
	}
}

// TestBackupHandlerConstruction verifies backup handler can be created
func TestBackupHandlerConstruction(t *testing.T) {
	handler := NewBackupHandler("", nil)

	if handler == nil {
		t.Error("NewBackupHandler should return non-nil handler")
	}
}

// TestSetupHandlerConstruction verifies setup handler can be created
func TestSetupHandlerConstruction(t *testing.T) {
	handler := NewSetupHandler(nil, "", nil)

	if handler == nil {
		t.Error("NewSetupHandler should return non-nil handler")
	}
}

// TestSetupAdminRejectsShortPassword verifies password minimum length is enforced
func TestSetupAdminRejectsShortPassword(t *testing.T) {
	handler := NewSetupHandler(nil, t.TempDir(), nil)

	// Create a session first by calling getSession through requireSession
	// We'll test the password validation directly by simulating a POST

	// We need templates for the handler to work, but since we're testing
	// a POST that returns an error before rendering, nil templates are fine
	// for the error path.

	shortPasswords := []string{"", "a", "1234", "short", "1234567"}
	for _, pw := range shortPasswords {
		form := strings.NewReader("username=admin&password=" + pw + "&confirm_password=" + pw)
		req := httptest.NewRequest(http.MethodPost, "/setup/admin", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// Add a fake session cookie so getSession returns one
		req.AddCookie(&http.Cookie{Name: "wizard_session", Value: "test-session"})
		w := httptest.NewRecorder()

		handler.HandleAdmin(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("password %q (len %d): expected status 400, got %d", pw, len(pw), w.Code)
		}
	}
}

// TestSetupAdminAcceptsValidPassword verifies valid passwords are accepted
func TestSetupAdminAcceptsValidPassword(t *testing.T) {
	handler := NewSetupHandler(nil, t.TempDir(), nil)

	validPasswords := []string{"12345678", "MyP@ssw0rd!", "a-very-long-and-secure-password"}
	for _, pw := range validPasswords {
		form := strings.NewReader("username=admin&password=" + pw + "&confirm_password=" + pw)
		req := httptest.NewRequest(http.MethodPost, "/setup/admin", form)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.AddCookie(&http.Cookie{Name: "wizard_session", Value: "test-valid-pw"})
		w := httptest.NewRecorder()

		handler.HandleAdmin(w, req)

		// Should NOT be 400 - it may be 200 (redirect) or 500 (template nil)
		// but critically it must not be 400 (validation rejection)
		if w.Code == http.StatusBadRequest {
			t.Errorf("password %q (len %d): should be accepted, got 400", pw, len(pw))
		}
	}
}

// TestGenerateSessionIDReturnsUniqueValues verifies session IDs are unique
func TestGenerateSessionIDReturnsUniqueValues(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := generateSessionID()
		if err != nil {
			t.Fatalf("generateSessionID failed: %v", err)
		}
		if id == "" {
			t.Fatal("session ID should not be empty")
		}
		if ids[id] {
			t.Fatalf("duplicate session ID generated: %s", id)
		}
		ids[id] = true
	}
}

// TestGenerateSessionIDLength verifies session IDs have expected length
func TestGenerateSessionIDLength(t *testing.T) {
	id, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID failed: %v", err)
	}
	// 16 bytes base64url encoded = 22 chars (no padding with RawURLEncoding)
	if len(id) != 22 {
		t.Errorf("expected session ID length 22, got %d", len(id))
	}
}

// TestCheckWebSocketOriginSameOrigin verifies same-origin connections are allowed
func TestCheckWebSocketOriginSameOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/logs/plex/stream", nil)
	req.Host = "localhost:3000"
	req.Header.Set("Origin", "http://localhost:3000")

	if !checkWebSocketOrigin(req) {
		t.Error("same-origin request should be allowed")
	}
}

// TestCheckWebSocketOriginNoOrigin verifies requests without Origin are allowed (non-browser)
func TestCheckWebSocketOriginNoOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/logs/plex/stream", nil)
	req.Host = "localhost:3000"

	if !checkWebSocketOrigin(req) {
		t.Error("request without Origin header should be allowed")
	}
}

// TestCheckWebSocketOriginCrossOriginRejected verifies cross-origin connections are blocked
func TestCheckWebSocketOriginCrossOriginRejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/logs/plex/stream", nil)
	req.Host = "localhost:3000"
	req.Header.Set("Origin", "http://evil.com")

	if checkWebSocketOrigin(req) {
		t.Error("cross-origin request should be rejected")
	}
}

// TestCheckWebSocketOriginDifferentPort verifies different port is rejected
func TestCheckWebSocketOriginDifferentPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/logs/plex/stream", nil)
	req.Host = "localhost:3000"
	req.Header.Set("Origin", "http://localhost:4000")

	if checkWebSocketOrigin(req) {
		t.Error("different port in origin should be rejected")
	}
}

// TestCheckWebSocketOriginHTTPSMatch verifies HTTPS same-host matching works
func TestCheckWebSocketOriginHTTPSMatch(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/logs/plex/stream", nil)
	req.Host = "sdbx.example.com"
	req.Header.Set("Origin", "https://sdbx.example.com")

	if !checkWebSocketOrigin(req) {
		t.Error("same-host HTTPS origin should be allowed")
	}
}

// TestCheckWebSocketOriginInvalidURL verifies malformed origin is rejected
func TestCheckWebSocketOriginInvalidURL(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/logs/plex/stream", nil)
	req.Host = "localhost:3000"
	req.Header.Set("Origin", "://invalid")

	if checkWebSocketOrigin(req) {
		t.Error("malformed origin should be rejected")
	}
}
