package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maiko/sdbx/internal/config"
)

// TestConfigHandlerValidateConfig tests the validate endpoint
func TestConfigHandlerValidateConfig(t *testing.T) {
	handler := NewConfigHandler("", nil)

	tests := []struct {
		name       string
		content    string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "empty content",
			content:    "",
			wantStatus: http.StatusBadRequest,
			wantBody:   "Config content is required",
		},
		{
			name:       "invalid YAML",
			content:    "invalid: yaml: syntax:",
			wantStatus: http.StatusBadRequest,
			wantBody:   "Invalid YAML syntax",
		},
		{
			name: "valid config",
			content: `domain: example.com
timezone: UTC
expose:
  mode: cloudflared
routing:
  strategy: subdomain
mediapath: ./data/media
downloadspath: ./data/downloads
configpath: ./configs`,
			wantStatus: http.StatusOK,
			wantBody:   "Config is valid",
		},
		{
			name: "missing domain",
			content: `timezone: UTC
expose:
  mode: cloudflared
routing:
  strategy: subdomain
mediapath: ./data/media
downloadspath: ./data/downloads
configpath: ./configs`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "domain is required",
		},
		{
			name: "invalid expose mode",
			content: `domain: example.com
timezone: UTC
expose:
  mode: invalid
routing:
  strategy: subdomain
mediapath: ./data/media
downloadspath: ./data/downloads
configpath: ./configs`,
			wantStatus: http.StatusBadRequest,
			wantBody:   "expose.mode must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("content", tt.content)

			req := httptest.NewRequest(http.MethodPost, "/api/config/validate", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			handler.HandleValidateConfig(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			body := w.Body.String()
			if !strings.Contains(body, tt.wantBody) {
				t.Errorf("body = %q, want to contain %q", body, tt.wantBody)
			}
		})
	}
}

// TestConfigHandlerGetConfig tests the get config endpoint
func TestConfigHandlerGetConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test config file
	configContent := `domain: test.example.com
timezone: America/New_York`
	configPath := filepath.Join(tmpDir, ".sdbx.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create test config: %v", err)
	}

	handler := NewConfigHandler(tmpDir, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	handler.HandleGetConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body != configContent {
		t.Errorf("body = %q, want %q", body, configContent)
	}

	// Verify content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain" {
		t.Errorf("Content-Type = %q, want text/plain", contentType)
	}
}

// TestConfigHandlerGetConfigNotFound tests get config when file doesn't exist
func TestConfigHandlerGetConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewConfigHandler(tmpDir, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	w := httptest.NewRecorder()

	handler.HandleGetConfig(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Failed to read config") {
		t.Errorf("body = %q, want to contain 'Failed to read config'", body)
	}
}

// TestConfigHandlerSaveConfig tests the save config endpoint
func TestConfigHandlerSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an existing config
	existingConfig := "domain: old.example.com"
	configPath := filepath.Join(tmpDir, ".sdbx.yaml")
	if err := os.WriteFile(configPath, []byte(existingConfig), 0644); err != nil {
		t.Fatalf("failed to create existing config: %v", err)
	}

	handler := NewConfigHandler(tmpDir, nil)

	newConfig := `domain: new.example.com
timezone: UTC
expose:
  mode: cloudflared
routing:
  strategy: subdomain
mediapath: ./data/media
downloadspath: ./data/downloads
configpath: ./configs`

	form := url.Values{}
	form.Set("content", newConfig)

	req := httptest.NewRequest(http.MethodPost, "/api/config/save", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleSaveConfig(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	// Verify file was updated
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read saved config: %v", err)
	}

	if string(content) != newConfig {
		t.Errorf("saved config = %q, want %q", string(content), newConfig)
	}

	// Verify backup was cleaned up
	backupPath := configPath + ".backup"
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("backup file should be removed after successful save")
	}
}

// TestConfigHandlerSaveConfigEmpty tests save with empty content
func TestConfigHandlerSaveConfigEmpty(t *testing.T) {
	handler := NewConfigHandler("", nil)

	form := url.Values{}
	form.Set("content", "")

	req := httptest.NewRequest(http.MethodPost, "/api/config/save", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleSaveConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// TestConfigHandlerSaveConfigInvalidYAML tests save with invalid YAML
func TestConfigHandlerSaveConfigInvalidYAML(t *testing.T) {
	handler := NewConfigHandler("", nil)

	form := url.Values{}
	form.Set("content", "invalid: yaml: :")

	req := httptest.NewRequest(http.MethodPost, "/api/config/save", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.HandleSaveConfig(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Invalid YAML syntax") {
		t.Errorf("body = %q, want to contain 'Invalid YAML syntax'", body)
	}
}

// TestValidateConfig tests the internal validation function
func TestValidateConfig(t *testing.T) {
	handler := NewConfigHandler("", nil)

	tests := []struct {
		name       string
		cfg        *config.Config
		wantErrors []string
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				Domain:        "example.com",
				Timezone:      "UTC",
				Expose:        config.ExposeConfig{Mode: "cloudflared"},
				Routing:       config.RoutingConfig{Strategy: "subdomain"},
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				ConfigPath:    "./configs",
			},
			wantErrors: nil,
		},
		{
			name: "missing domain",
			cfg: &config.Config{
				Domain:        "",
				Timezone:      "UTC",
				Expose:        config.ExposeConfig{Mode: "cloudflared"},
				Routing:       config.RoutingConfig{Strategy: "subdomain"},
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				ConfigPath:    "./configs",
			},
			wantErrors: []string{"domain is required"},
		},
		{
			name: "missing timezone",
			cfg: &config.Config{
				Domain:        "example.com",
				Timezone:      "",
				Expose:        config.ExposeConfig{Mode: "cloudflared"},
				Routing:       config.RoutingConfig{Strategy: "subdomain"},
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				ConfigPath:    "./configs",
			},
			wantErrors: []string{"timezone is required"},
		},
		{
			name: "invalid expose mode",
			cfg: &config.Config{
				Domain:        "example.com",
				Timezone:      "UTC",
				Expose:        config.ExposeConfig{Mode: "invalid"},
				Routing:       config.RoutingConfig{Strategy: "subdomain"},
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				ConfigPath:    "./configs",
			},
			wantErrors: []string{"expose.mode must be one of"},
		},
		{
			name: "invalid routing strategy",
			cfg: &config.Config{
				Domain:        "example.com",
				Timezone:      "UTC",
				Expose:        config.ExposeConfig{Mode: "cloudflared"},
				Routing:       config.RoutingConfig{Strategy: "invalid"},
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				ConfigPath:    "./configs",
			},
			wantErrors: []string{"routing.strategy must be one of"},
		},
		{
			name: "missing paths",
			cfg: &config.Config{
				Domain:        "example.com",
				Timezone:      "UTC",
				Expose:        config.ExposeConfig{Mode: "cloudflared"},
				Routing:       config.RoutingConfig{Strategy: "subdomain"},
				MediaPath:     "",
				DownloadsPath: "",
				ConfigPath:    "",
			},
			wantErrors: []string{"media_path is required", "downloads_path is required", "config_path is required"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := handler.validateConfig(tt.cfg)

			if tt.wantErrors == nil {
				if len(errors) != 0 {
					t.Errorf("expected no errors, got %v", errors)
				}
				return
			}

			for _, wantErr := range tt.wantErrors {
				found := false
				for _, err := range errors {
					if strings.Contains(err, wantErr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, got %v", wantErr, errors)
				}
			}
		})
	}
}

// TestRespondJSON tests JSON response helper
func TestRespondJSON(t *testing.T) {
	handler := NewConfigHandler("", nil)

	w := httptest.NewRecorder()
	data := ConfigResponse{
		Success: true,
		Message: "test message",
	}

	handler.respondJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "test message") {
		t.Errorf("body = %q, want to contain 'test message'", body)
	}
}
