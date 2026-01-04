package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Domain != "sdbx.example.com" {
		t.Errorf("Domain = %s, want sdbx.example.com", cfg.Domain)
	}
	if cfg.Expose.Mode != "cloudflared" {
		t.Errorf("Expose.Mode = %s, want cloudflared", cfg.Expose.Mode)
	}
	if cfg.PUID != 1000 {
		t.Errorf("PUID = %d, want 1000", cfg.PUID)
	}
	if cfg.PGID != 1000 {
		t.Errorf("PGID = %d, want 1000", cfg.PGID)
	}
}

func TestAddonManagement(t *testing.T) {
	cfg := DefaultConfig()

	// Initially no addons
	if len(cfg.Addons) != 0 {
		t.Errorf("Initial addons count = %d, want 0", len(cfg.Addons))
	}

	// Enable addon
	cfg.EnableAddon("overseerr")
	if !cfg.IsAddonEnabled("overseerr") {
		t.Error("overseerr should be enabled")
	}
	if len(cfg.Addons) != 1 {
		t.Errorf("Addons count = %d, want 1", len(cfg.Addons))
	}

	// Enable same addon again (should not duplicate)
	cfg.EnableAddon("overseerr")
	if len(cfg.Addons) != 1 {
		t.Errorf("Addons count = %d after duplicate enable, want 1", len(cfg.Addons))
	}

	// Enable another addon
	cfg.EnableAddon("wizarr")
	if !cfg.IsAddonEnabled("wizarr") {
		t.Error("wizarr should be enabled")
	}
	if len(cfg.Addons) != 2 {
		t.Errorf("Addons count = %d, want 2", len(cfg.Addons))
	}

	// Disable addon
	cfg.DisableAddon("overseerr")
	if cfg.IsAddonEnabled("overseerr") {
		t.Error("overseerr should be disabled")
	}
	if len(cfg.Addons) != 1 {
		t.Errorf("Addons count = %d after disable, want 1", len(cfg.Addons))
	}

	// wizarr should still be enabled
	if !cfg.IsAddonEnabled("wizarr") {
		t.Error("wizarr should still be enabled")
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested directory
	nested := filepath.Join(tmpDir, "a", "b", "c")
	if err := EnsureDir(nested); err != nil {
		t.Fatalf("EnsureDir failed: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(nested); os.IsNotExist(err) {
		t.Error("EnsureDir did not create directory")
	}

	// Create again should not fail
	if err := EnsureDir(nested); err != nil {
		t.Errorf("EnsureDir failed on existing dir: %v", err)
	}
}

func TestProjectDir(t *testing.T) {
	// Save current dir
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	// Create temp project
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// No project files - should fail
	os.Chdir(tmpDir)
	_, err = ProjectDir()
	if err == nil {
		t.Error("ProjectDir should fail without project files")
	}

	// Create .sdbx.yaml
	os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: test.com"), 0o644)

	// Now should succeed
	dir, err := ProjectDir()
	if err != nil {
		t.Errorf("ProjectDir failed: %v", err)
	}

	// Resolve symlinks for comparison (macOS /var -> /private/var)
	expectedDir, _ := filepath.EvalSymlinks(tmpDir)
	actualDir, _ := filepath.EvalSymlinks(dir)
	if actualDir != expectedDir {
		t.Errorf("ProjectDir = %s, want %s", actualDir, expectedDir)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errField string
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "missing domain",
			config: &Config{
				Timezone: "UTC",
				Expose:   ExposeConfig{Mode: "cloudflared"},
				Routing:  RoutingConfig{Strategy: "subdomain"},
				ConfigPath:    "./config",
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				PUID:          1000,
				PGID:          1000,
			},
			wantErr:  true,
			errField: "domain",
		},
		{
			name: "invalid domain format",
			config: &Config{
				Domain:   "invalid_domain",
				Timezone: "UTC",
				Expose:   ExposeConfig{Mode: "cloudflared"},
				Routing:  RoutingConfig{Strategy: "subdomain"},
				ConfigPath:    "./config",
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				PUID:          1000,
				PGID:          1000,
			},
			wantErr:  true,
			errField: "domain",
		},
		{
			name: "invalid expose mode",
			config: &Config{
				Domain:   "sdbx.example.com",
				Timezone: "UTC",
				Expose:   ExposeConfig{Mode: "invalid"},
				Routing:  RoutingConfig{Strategy: "subdomain"},
				ConfigPath:    "./config",
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				PUID:          1000,
				PGID:          1000,
			},
			wantErr:  true,
			errField: "expose.mode",
		},
		{
			name: "invalid routing strategy",
			config: &Config{
				Domain:   "sdbx.example.com",
				Timezone: "UTC",
				Expose:   ExposeConfig{Mode: "cloudflared"},
				Routing:  RoutingConfig{Strategy: "invalid"},
				ConfigPath:    "./config",
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				PUID:          1000,
				PGID:          1000,
			},
			wantErr:  true,
			errField: "routing.strategy",
		},
		{
			name: "path routing without base domain",
			config: &Config{
				Domain:   "sdbx.example.com",
				Timezone: "UTC",
				Expose:   ExposeConfig{Mode: "cloudflared"},
				Routing:  RoutingConfig{Strategy: "path", BaseDomain: ""},
				ConfigPath:    "./config",
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				PUID:          1000,
				PGID:          1000,
			},
			wantErr:  true,
			errField: "routing.base_domain",
		},
		{
			name: "vpn enabled without provider",
			config: &Config{
				Domain:     "sdbx.example.com",
				Timezone:   "UTC",
				Expose:     ExposeConfig{Mode: "cloudflared"},
				Routing:    RoutingConfig{Strategy: "subdomain"},
				VPNEnabled: true,
				ConfigPath:    "./config",
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				PUID:          1000,
				PGID:          1000,
			},
			wantErr:  true,
			errField: "vpn_provider",
		},
		{
			name: "invalid PUID",
			config: &Config{
				Domain:   "sdbx.example.com",
				Timezone: "UTC",
				Expose:   ExposeConfig{Mode: "cloudflared"},
				Routing:  RoutingConfig{Strategy: "subdomain"},
				ConfigPath:    "./config",
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				PUID:          -1,
				PGID:          1000,
			},
			wantErr:  true,
			errField: "puid",
		},
		{
			name: "invalid PGID",
			config: &Config{
				Domain:   "sdbx.example.com",
				Timezone: "UTC",
				Expose:   ExposeConfig{Mode: "cloudflared"},
				Routing:  RoutingConfig{Strategy: "subdomain"},
				ConfigPath:    "./config",
				MediaPath:     "./media",
				DownloadsPath: "./downloads",
				PUID:          1000,
				PGID:          70000,
			},
			wantErr:  true,
			errField: "pgid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				// Check if it's a ValidationError with the correct field
				if valErr, ok := err.(*ValidationError); ok {
					if valErr.Field != tt.errField {
						t.Errorf("Validate() error field = %s, want %s", valErr.Field, tt.errField)
					}
				}
			}
		})
	}
}

func TestProjectNotFoundError(t *testing.T) {
	err := &ProjectNotFoundError{StartPath: "/test/path"}
	if !IsProjectNotFoundError(err) {
		t.Error("IsProjectNotFoundError should return true for ProjectNotFoundError")
	}

	if IsProjectNotFoundError(nil) {
		t.Error("IsProjectNotFoundError should return false for nil")
	}

	genericErr := fmt.Errorf("generic error")
	if IsProjectNotFoundError(genericErr) {
		t.Error("IsProjectNotFoundError should return false for generic error")
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("test_field", "test message")
	expectedMsg := "validation error [test_field]: test message"
	if err.Error() != expectedMsg {
		t.Errorf("ValidationError.Error() = %s, want %s", err.Error(), expectedMsg)
	}
}

func TestGetServiceRoutingStrategy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Routing.Strategy = "subdomain"

	// Default: use global strategy
	if cfg.GetServiceRoutingStrategy("sonarr") != "subdomain" {
		t.Error("should return global strategy for service without override")
	}

	// With override
	cfg.Services["radarr"] = ServiceOverride{Routing: "path"}
	if cfg.GetServiceRoutingStrategy("radarr") != "path" {
		t.Error("should return override strategy")
	}

	// Empty override should fall back to global
	cfg.Services["lidarr"] = ServiceOverride{Routing: ""}
	if cfg.GetServiceRoutingStrategy("lidarr") != "subdomain" {
		t.Error("should return global strategy for empty override")
	}
}

func TestGetServiceSubdomain(t *testing.T) {
	cfg := DefaultConfig()

	// Default: use service name
	if cfg.GetServiceSubdomain("sonarr") != "sonarr" {
		t.Error("should return service name as subdomain")
	}

	// With override
	cfg.Services["overseerr"] = ServiceOverride{Subdomain: "requests"}
	if cfg.GetServiceSubdomain("overseerr") != "requests" {
		t.Error("should return custom subdomain")
	}
}

func TestGetServicePath(t *testing.T) {
	cfg := DefaultConfig()

	// Default: /service-name
	if cfg.GetServicePath("sonarr") != "/sonarr" {
		t.Error("should return /service-name as path")
	}

	// With override
	cfg.Services["radarr"] = ServiceOverride{Path: "/movies"}
	if cfg.GetServicePath("radarr") != "/movies" {
		t.Error("should return custom path")
	}
}

func TestGetServiceURL(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Domain = "example.com"
	cfg.Routing.Strategy = "subdomain"

	// Subdomain routing
	url := cfg.GetServiceURL("sonarr")
	if url != "https://sonarr.example.com" {
		t.Errorf("subdomain URL = %s, want https://sonarr.example.com", url)
	}

	// Path routing
	cfg.Routing.Strategy = "path"
	cfg.Routing.BaseDomain = "apps"
	url = cfg.GetServiceURL("radarr")
	if url != "https://apps.example.com/radarr" {
		t.Errorf("path URL = %s, want https://apps.example.com/radarr", url)
	}

	// Path routing without base domain
	cfg.Routing.BaseDomain = ""
	url = cfg.GetServiceURL("lidarr")
	if url != "https://example.com/lidarr" {
		t.Errorf("path URL without base = %s, want https://example.com/lidarr", url)
	}

	// Custom subdomain override
	cfg.Routing.Strategy = "subdomain"
	cfg.Services["overseerr"] = ServiceOverride{Subdomain: "requests"}
	url = cfg.GetServiceURL("overseerr")
	if url != "https://requests.example.com" {
		t.Errorf("custom subdomain URL = %s, want https://requests.example.com", url)
	}
}

func TestIsPathRouting(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Routing.Strategy = "subdomain"

	if cfg.IsPathRouting("sonarr") {
		t.Error("should return false for subdomain routing")
	}

	cfg.Routing.Strategy = "path"
	if !cfg.IsPathRouting("sonarr") {
		t.Error("should return true for path routing")
	}

	// Service override
	cfg.Services["radarr"] = ServiceOverride{Routing: "subdomain"}
	if cfg.IsPathRouting("radarr") {
		t.Error("should return false for service with subdomain override")
	}
}

func TestNeedsTLS(t *testing.T) {
	cfg := DefaultConfig()

	cfg.Expose.Mode = ExposeModeDirect
	if !cfg.NeedsTLS() {
		t.Error("direct mode should need TLS")
	}

	cfg.Expose.Mode = ExposeModeCloudflared
	if cfg.NeedsTLS() {
		t.Error("cloudflared mode should not need TLS")
	}

	cfg.Expose.Mode = ExposeModeLAN
	if cfg.NeedsTLS() {
		t.Error("lan mode should not need TLS")
	}
}

func TestIsCloudflared(t *testing.T) {
	cfg := DefaultConfig()

	cfg.Expose.Mode = ExposeModeCloudflared
	if !cfg.IsCloudflared() {
		t.Error("should return true for cloudflared mode")
	}

	cfg.Expose.Mode = ExposeModeDirect
	if cfg.IsCloudflared() {
		t.Error("should return false for direct mode")
	}
}

func TestIsLANMode(t *testing.T) {
	cfg := DefaultConfig()

	cfg.Expose.Mode = ExposeModeLAN
	if !cfg.IsLANMode() {
		t.Error("should return true for lan mode")
	}

	cfg.Expose.Mode = ExposeModeCloudflared
	if cfg.IsLANMode() {
		t.Error("should return false for cloudflared mode")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sdbx.yaml")

	cfg := DefaultConfig()
	cfg.Domain = "test.example.com"
	cfg.Addons = []string{"sonarr", "radarr"}

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file should exist after save")
	}
}

func TestTimezoneValidation(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		wantErr  bool
	}{
		{"valid UTC", "UTC", false},
		{"valid America/New_York", "America/New_York", false},
		{"valid Europe/London", "Europe/London", false},
		{"valid Asia/Tokyo", "Asia/Tokyo", false},
		{"invalid timezone", "Invalid/Timezone", true},
		{"empty timezone", "", true},
		{"random string", "not-a-timezone", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Timezone = tt.timezone

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() with timezone %q error = %v, wantErr %v", tt.timezone, err, tt.wantErr)
			}
		})
	}
}

func TestContainsHelper(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !contains(slice, "a") {
		t.Error("should find 'a' in slice")
	}

	if !contains(slice, "c") {
		t.Error("should find 'c' in slice")
	}

	if contains(slice, "d") {
		t.Error("should not find 'd' in slice")
	}

	if contains([]string{}, "a") {
		t.Error("should not find anything in empty slice")
	}
}

func TestDomainValidation(t *testing.T) {
	tests := []struct {
		domain  string
		wantErr bool
	}{
		{"example.com", false},
		{"sub.example.com", false},
		{"a.b.c.example.com", false},
		{"my-domain.io", false},
		{"example123.com", false},
		{"", true},
		{"invalid", true},
		{"invalid_domain.com", true},
		{"-invalid.com", true},
		{"invalid-.com", true},
		{".com", true},
		{"example.", true},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Domain = tt.domain

			err := cfg.Validate()
			hasErr := err != nil
			if hasErr != tt.wantErr {
				t.Errorf("domain %q: error = %v, wantErr = %v", tt.domain, err, tt.wantErr)
			}
		})
	}
}
