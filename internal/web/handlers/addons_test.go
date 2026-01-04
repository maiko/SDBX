package handlers

import (
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
