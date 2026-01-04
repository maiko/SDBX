package handlers

import (
	"testing"
)

// TestCountRunningServices verifies counting running services
func TestCountRunningServices(t *testing.T) {
	tests := []struct {
		name     string
		services map[string]ServiceInfo
		want     int
	}{
		{
			name:     "empty map",
			services: map[string]ServiceInfo{},
			want:     0,
		},
		{
			name: "all stopped",
			services: map[string]ServiceInfo{
				"sonarr":  {Name: "sonarr", Running: false},
				"radarr":  {Name: "radarr", Running: false},
				"traefik": {Name: "traefik", Running: false},
			},
			want: 0,
		},
		{
			name: "all running",
			services: map[string]ServiceInfo{
				"sonarr":  {Name: "sonarr", Running: true},
				"radarr":  {Name: "radarr", Running: true},
				"traefik": {Name: "traefik", Running: true},
			},
			want: 3,
		},
		{
			name: "mixed",
			services: map[string]ServiceInfo{
				"sonarr":  {Name: "sonarr", Running: true},
				"radarr":  {Name: "radarr", Running: false},
				"traefik": {Name: "traefik", Running: true},
				"plex":    {Name: "plex", Running: false},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countRunningServices(tt.services)
			if got != tt.want {
				t.Errorf("countRunningServices() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestServiceInfoStruct verifies service info struct
func TestServiceInfoStruct(t *testing.T) {
	info := ServiceInfo{
		Name:        "sonarr",
		DisplayName: "Sonarr",
		Status:      "running",
		Health:      "healthy",
		Running:     true,
		Category:    "media",
		Description: "TV Shows automation",
		URL:         "https://sonarr.example.com",
		HasWebUI:    true,
	}

	if info.Name != "sonarr" {
		t.Errorf("Name = %q, want 'sonarr'", info.Name)
	}

	if info.Status != "running" {
		t.Errorf("Status = %q, want 'running'", info.Status)
	}

	if !info.Running {
		t.Error("Running should be true")
	}

	if !info.HasWebUI {
		t.Error("HasWebUI should be true")
	}
}

// TestServiceInfoWithDefaultValues verifies defaults
func TestServiceInfoWithDefaultValues(t *testing.T) {
	info := ServiceInfo{
		Name: "test-service",
	}

	if info.Running {
		t.Error("Running should default to false")
	}

	if info.HasWebUI {
		t.Error("HasWebUI should default to false")
	}

	if info.Status != "" {
		t.Error("Status should default to empty")
	}
}
