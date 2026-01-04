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

// TestIntegrationHandlerConstruction verifies integration handler can be created
func TestIntegrationHandlerConstruction(t *testing.T) {
	handler := NewIntegrationHandler("", nil)

	if handler == nil {
		t.Error("NewIntegrationHandler should return non-nil handler")
	}
}

// TestSetupHandlerConstruction verifies setup handler can be created
func TestSetupHandlerConstruction(t *testing.T) {
	handler := NewSetupHandler(nil, "", nil)

	if handler == nil {
		t.Error("NewSetupHandler should return non-nil handler")
	}
}
