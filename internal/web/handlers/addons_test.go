package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCountEnabledAddons verifies addon counting
func TestCountEnabledAddons(t *testing.T) {
	tests := []struct {
		name   string
		addons []AddonDisplay
		want   int
	}{
		{
			name:   "empty list",
			addons: []AddonDisplay{},
			want:   0,
		},
		{
			name: "all disabled",
			addons: []AddonDisplay{
				{Name: "sonarr", Enabled: false},
				{Name: "radarr", Enabled: false},
			},
			want: 0,
		},
		{
			name: "all enabled",
			addons: []AddonDisplay{
				{Name: "sonarr", Enabled: true},
				{Name: "radarr", Enabled: true},
			},
			want: 2,
		},
		{
			name: "mixed",
			addons: []AddonDisplay{
				{Name: "sonarr", Enabled: true},
				{Name: "radarr", Enabled: false},
				{Name: "prowlarr", Enabled: true},
				{Name: "lidarr", Enabled: false},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countEnabledAddons(tt.addons)
			if got != tt.want {
				t.Errorf("countEnabledAddons() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestAddonDisplayStruct verifies addon display struct
func TestAddonDisplayStruct(t *testing.T) {
	addon := AddonDisplay{
		Name:        "sonarr",
		DisplayName: "Sonarr",
		Description: "TV Shows automation",
		Category:    "media",
		Version:     "1.0.0",
		Source:      "official",
		Enabled:     true,
		HasWebUI:    true,
	}

	if addon.Name != "sonarr" {
		t.Errorf("Name = %q, want 'sonarr'", addon.Name)
	}

	if !addon.Enabled {
		t.Error("Enabled should be true")
	}

	if !addon.HasWebUI {
		t.Error("HasWebUI should be true")
	}
}

// TestHandleEnableAddonMissingName verifies enable requires addon name
func TestHandleEnableAddonMissingName(t *testing.T) {
	handler := NewAddonsHandler(nil, "", nil)

	req := httptest.NewRequest(http.MethodPost, "/api/addons//enable", nil)
	w := httptest.NewRecorder()

	handler.HandleEnableAddon(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Addon name is required") {
		t.Errorf("body = %q, want to contain 'Addon name is required'", body)
	}
}

// TestHandleDisableAddonMissingName verifies disable requires addon name
func TestHandleDisableAddonMissingName(t *testing.T) {
	handler := NewAddonsHandler(nil, "", nil)

	req := httptest.NewRequest(http.MethodPost, "/api/addons//disable", nil)
	w := httptest.NewRecorder()

	handler.HandleDisableAddon(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Addon name is required") {
		t.Errorf("body = %q, want to contain 'Addon name is required'", body)
	}
}

// TestAddonResponseStruct verifies addon response struct
func TestAddonResponseStruct(t *testing.T) {
	resp := AddonResponse{
		Success: true,
		Message: "Addon enabled successfully",
		Addon:   "sonarr",
	}

	if !resp.Success {
		t.Error("Success should be true")
	}

	if resp.Addon != "sonarr" {
		t.Errorf("Addon = %q, want 'sonarr'", resp.Addon)
	}
}

// TestAddonResponsePendingRestart verifies pendingRestart field in addon response
func TestAddonResponsePendingRestart(t *testing.T) {
	resp := AddonResponse{
		Success:        true,
		Message:        "Enabled 'sonarr'. Run 'sdbx up' to start the service.",
		Addon:          "sonarr",
		PendingRestart: true,
	}

	if !resp.PendingRestart {
		t.Error("PendingRestart should be true after enable/disable")
	}

	respNoPending := AddonResponse{
		Success: true,
		Message: "Already enabled",
		Addon:   "sonarr",
	}

	if respNoPending.PendingRestart {
		t.Error("PendingRestart should be false when not set")
	}
}

// TestValidLogServiceName verifies log service name validation regex
func TestValidLogServiceName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"valid simple", "sonarr", true},
		{"valid with hyphen", "my-service", true},
		{"valid with underscore", "my_service", true},
		{"valid with numbers", "service123", true},
		{"starts with number", "1service", true},
		{"empty", "", false},
		{"starts with hyphen", "-service", false},
		{"starts with underscore", "_service", false},
		{"uppercase", "Sonarr", false},
		{"spaces", "my service", false},
		{"special chars", "service;rm", false},
		{"path traversal", "../etc/passwd", false},
		{"too long", "a" + strings.Repeat("b", 64), false},
		{"max length", "a" + strings.Repeat("b", 63), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validLogServiceName.MatchString(tt.input)
			if got != tt.valid {
				t.Errorf("validLogServiceName.MatchString(%q) = %v, want %v", tt.input, got, tt.valid)
			}
		})
	}
}
