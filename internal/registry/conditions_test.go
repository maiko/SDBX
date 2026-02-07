package registry

import (
	"testing"

	"github.com/maiko/sdbx/internal/config"
)

func TestEvaluateConditionsAlways(t *testing.T) {
	cfg := &config.Config{}
	cond := Conditions{Always: true}

	if !EvaluateConditions(cond, cfg) {
		t.Error("always=true should return true regardless of config")
	}
}

func TestEvaluateConditionsVPNEnabled(t *testing.T) {
	tests := []struct {
		name       string
		vpnEnabled bool
		expected   bool
	}{
		{"VPN enabled", true, true},
		{"VPN disabled", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{VPNEnabled: tt.vpnEnabled}
			cond := Conditions{RequireConfig: "vpn_enabled"}

			if got := EvaluateConditions(cond, cfg); got != tt.expected {
				t.Errorf("EvaluateConditions() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEvaluateConditionsCloudflared(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected bool
	}{
		{"cloudflared mode", config.ExposeModeCloudflared, true},
		{"lan mode", config.ExposeModeLAN, false},
		{"direct mode", config.ExposeModeDirect, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cfg.Expose.Mode = tt.mode
			cond := Conditions{RequireConfig: "cloudflared"}

			if got := EvaluateConditions(cond, cfg); got != tt.expected {
				t.Errorf("EvaluateConditions() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEvaluateConditionsNoConditions(t *testing.T) {
	cfg := &config.Config{}
	cond := Conditions{}

	if !EvaluateConditions(cond, cfg) {
		t.Error("empty conditions should return true")
	}
}

func TestEvaluateConditionsRequireAddonIgnored(t *testing.T) {
	cfg := &config.Config{}
	cond := Conditions{RequireAddon: true}

	// RequireAddon is handled by the resolver, not EvaluateConditions
	if !EvaluateConditions(cond, cfg) {
		t.Error("requireAddon should not cause EvaluateConditions to return false")
	}
}

func TestEvaluateConditionsUnknownConfig(t *testing.T) {
	cfg := &config.Config{}
	cond := Conditions{RequireConfig: "unknown_feature"}

	// Unknown config conditions should not prevent inclusion
	if !EvaluateConditions(cond, cfg) {
		t.Error("unknown requireConfig should return true (fail open)")
	}
}
