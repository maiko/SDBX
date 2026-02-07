package cmd

import (
	"testing"
)

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"sdbx-radarr", "radarr"},
		{"sdbx-sonarr", "sonarr"},
		{"sdbx-qbittorrent", "qbittorrent"},
		{"sdbx-sdbx-webui", "sdbx-webui"},
		{"standalone", "standalone"},
		{"project-my-service", "my-service"},
		{"", ""},
	}

	for _, tt := range tests {
		result := extractServiceName(tt.input)
		if result != tt.expected {
			t.Errorf("extractServiceName(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
