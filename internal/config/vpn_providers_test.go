package config

import (
	"testing"
)

func TestGetVPNProvider(t *testing.T) {
	tests := []struct {
		id      string
		exists  bool
		name    string
		authType VPNAuthType
	}{
		{"nordvpn", true, "NordVPN", VPNAuthWireguard},
		{"mullvad", true, "Mullvad", VPNAuthToken},
		{"protonvpn", true, "ProtonVPN", VPNAuthUserPass},
		{"pia", true, "Private Internet Access (PIA)", VPNAuthUserPass},
		{"ivpn", true, "IVPN", VPNAuthToken},
		{"airvpn", true, "AirVPN", VPNAuthToken},
		{"custom", true, "Custom/OpenVPN", VPNAuthConfig},
		{"invalid", false, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			provider, ok := GetVPNProvider(tt.id)
			if ok != tt.exists {
				t.Errorf("GetVPNProvider(%q) exists = %v, want %v", tt.id, ok, tt.exists)
			}
			if ok {
				if provider.Name != tt.name {
					t.Errorf("GetVPNProvider(%q).Name = %q, want %q", tt.id, provider.Name, tt.name)
				}
				if provider.AuthType != tt.authType {
					t.Errorf("GetVPNProvider(%q).AuthType = %q, want %q", tt.id, provider.AuthType, tt.authType)
				}
			}
		})
	}
}

func TestGetVPNProviderIDs(t *testing.T) {
	ids := GetVPNProviderIDs()

	// Check we have the expected number of providers
	expectedCount := 17
	if len(ids) != expectedCount {
		t.Errorf("GetVPNProviderIDs() returned %d providers, want %d", len(ids), expectedCount)
	}

	// Check that all IDs are valid
	for _, id := range ids {
		if _, ok := GetVPNProvider(id); !ok {
			t.Errorf("GetVPNProviderIDs() contains invalid ID %q", id)
		}
	}

	// Check that nordvpn is first (popular provider)
	if ids[0] != "nordvpn" {
		t.Errorf("GetVPNProviderIDs()[0] = %q, want nordvpn", ids[0])
	}

	// Check that custom is last
	if ids[len(ids)-1] != "custom" {
		t.Errorf("GetVPNProviderIDs() last element = %q, want custom", ids[len(ids)-1])
	}
}

func TestVPNProviderAuthTypes(t *testing.T) {
	// Test that all providers have valid auth types
	for id, provider := range VPNProviders {
		switch provider.AuthType {
		case VPNAuthUserPass, VPNAuthToken, VPNAuthWireguard, VPNAuthConfig:
			// Valid
		default:
			t.Errorf("Provider %q has invalid AuthType %q", id, provider.AuthType)
		}
	}
}

func TestVPNProviderProtocols(t *testing.T) {
	// Test that all providers support at least one protocol
	for id, provider := range VPNProviders {
		if !provider.SupportsWG && !provider.SupportsOpenVPN {
			t.Errorf("Provider %q supports neither WireGuard nor OpenVPN", id)
		}
	}
}

func TestVPNProviderHasCredDocs(t *testing.T) {
	// Test that all providers (except custom) have credential documentation URL
	for id, provider := range VPNProviders {
		if id != "custom" && provider.CredDocsURL == "" {
			t.Errorf("Provider %q is missing CredDocsURL", id)
		}
	}
}

func TestVPNProviderUserPassLabels(t *testing.T) {
	// Providers with UserPass auth should have username and password labels
	userPassProviders := []string{"protonvpn", "pia", "surfshark", "expressvpn", "windscribe", "ipvanish", "cyberghost", "torguard", "vyprvpn", "purevpn", "hidemyass", "perfectprivacy"}

	for _, id := range userPassProviders {
		provider, ok := GetVPNProvider(id)
		if !ok {
			t.Errorf("Provider %q not found", id)
			continue
		}
		if provider.AuthType != VPNAuthUserPass {
			t.Errorf("Provider %q should have VPNAuthUserPass, got %q", id, provider.AuthType)
		}
		if provider.UsernameLabel == "" {
			t.Errorf("Provider %q is missing UsernameLabel", id)
		}
		if provider.PasswordLabel == "" {
			t.Errorf("Provider %q is missing PasswordLabel", id)
		}
	}
}

func TestVPNProviderTokenLabels(t *testing.T) {
	// Providers with Token auth should have token label
	tokenProviders := []string{"mullvad", "ivpn", "airvpn"}

	for _, id := range tokenProviders {
		provider, ok := GetVPNProvider(id)
		if !ok {
			t.Errorf("Provider %q not found", id)
			continue
		}
		if provider.AuthType != VPNAuthToken {
			t.Errorf("Provider %q should have VPNAuthToken, got %q", id, provider.AuthType)
		}
		if provider.TokenLabel == "" {
			t.Errorf("Provider %q is missing TokenLabel", id)
		}
	}
}
