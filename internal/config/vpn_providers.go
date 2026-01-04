// Package config handles configuration loading and management for sdbx.
package config

// VPNAuthType defines what type of authentication a provider requires
type VPNAuthType string

const (
	// VPNAuthUserPass requires username and password
	VPNAuthUserPass VPNAuthType = "userpass"
	// VPNAuthToken requires an account number, token, or device key
	VPNAuthToken VPNAuthType = "token"
	// VPNAuthWireguard requires Wireguard private key and optionally address
	VPNAuthWireguard VPNAuthType = "wireguard"
	// VPNAuthConfig requires a custom OpenVPN config file
	VPNAuthConfig VPNAuthType = "config"
)

// VPNProvider defines the authentication requirements for a VPN provider
type VPNProvider struct {
	Name            string      // Display name
	ID              string      // Provider ID for gluetun
	AuthType        VPNAuthType // Primary auth type
	SupportsWG      bool        // Supports Wireguard
	SupportsOpenVPN bool        // Supports OpenVPN
	CredDocsURL     string      // URL to credential documentation
	UsernameLabel   string      // Label for username field (if applicable)
	PasswordLabel   string      // Label for password field (if applicable)
	TokenLabel      string      // Label for token field (if applicable)
	Notes           string      // Additional notes
}

// VPNProviders is the list of supported VPN providers with their auth requirements
var VPNProviders = map[string]VPNProvider{
	"nordvpn": {
		Name:            "NordVPN",
		ID:              "nordvpn",
		AuthType:        VPNAuthWireguard,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://my.nordaccount.com/dashboard/nordvpn/manual-configuration/",
		TokenLabel:      "Wireguard Private Key",
		UsernameLabel:   "Service Username (for OpenVPN)",
		PasswordLabel:   "Service Password (for OpenVPN)",
		Notes:           "Wireguard is recommended. Get the private key from NordVPN manual setup.",
	},
	"mullvad": {
		Name:            "Mullvad",
		ID:              "mullvad",
		AuthType:        VPNAuthToken,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://mullvad.net/en/account/",
		TokenLabel:      "Account Number",
		Notes:           "Use your 16-digit account number. Wireguard is recommended.",
	},
	"protonvpn": {
		Name:            "ProtonVPN",
		ID:              "protonvpn",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://account.protonvpn.com/account#openvpn",
		UsernameLabel:   "OpenVPN/IKEv2 Username",
		PasswordLabel:   "OpenVPN/IKEv2 Password",
		Notes:           "Use the OpenVPN credentials from your ProtonVPN dashboard, not your account password.",
	},
	"pia": {
		Name:            "Private Internet Access (PIA)",
		ID:              "private internet access",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://www.privateinternetaccess.com/pages/client-control-panel",
		UsernameLabel:   "Username",
		PasswordLabel:   "Password",
		Notes:           "Use your PIA account credentials.",
	},
	"surfshark": {
		Name:            "Surfshark",
		ID:              "surfshark",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://my.surfshark.com/vpn/manual-setup/main",
		UsernameLabel:   "Service Username",
		PasswordLabel:   "Service Password",
		Notes:           "Get service credentials from the manual setup page.",
	},
	"expressvpn": {
		Name:            "ExpressVPN",
		ID:              "expressvpn",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      false,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://www.expressvpn.com/setup#manual",
		UsernameLabel:   "Username",
		PasswordLabel:   "Password",
		Notes:           "Get OpenVPN credentials from the manual setup page.",
	},
	"windscribe": {
		Name:            "Windscribe",
		ID:              "windscribe",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://windscribe.com/getconfig/openvpn",
		UsernameLabel:   "Username",
		PasswordLabel:   "Password",
		Notes:           "Use OpenVPN credentials from the config generator page.",
	},
	"ipvanish": {
		Name:            "IPVanish",
		ID:              "ipvanish",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      false,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://account.ipvanish.com/",
		UsernameLabel:   "Username",
		PasswordLabel:   "Password",
		Notes:           "Use your IPVanish account credentials.",
	},
	"cyberghost": {
		Name:            "CyberGhost",
		ID:              "cyberghost",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://my.cyberghostvpn.com/",
		UsernameLabel:   "Username",
		PasswordLabel:   "Password",
		Notes:           "Get credentials from My Account > VPN > Configure Device.",
	},
	"ivpn": {
		Name:            "IVPN",
		ID:              "ivpn",
		AuthType:        VPNAuthToken,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://www.ivpn.net/account/",
		TokenLabel:      "Account ID",
		Notes:           "Use your IVPN account ID (starts with ivpn- or i-).",
	},
	"torguard": {
		Name:            "TorGuard",
		ID:              "torguard",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://torguard.net/clientarea.php",
		UsernameLabel:   "Username",
		PasswordLabel:   "Password",
		Notes:           "Use your TorGuard VPN credentials.",
	},
	"vyprvpn": {
		Name:            "VyprVPN",
		ID:              "vyprvpn",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://www.vyprvpn.com/account/",
		UsernameLabel:   "Email",
		PasswordLabel:   "Password",
		Notes:           "Use your VyprVPN account email and password.",
	},
	"purevpn": {
		Name:            "PureVPN",
		ID:              "purevpn",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      false,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://my.purevpn.com/",
		UsernameLabel:   "VPN Username",
		PasswordLabel:   "VPN Password",
		Notes:           "Get VPN credentials from the PureVPN member area.",
	},
	"hidemyass": {
		Name:            "HideMyAss (HMA)",
		ID:              "hidemyass",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      false,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://my.hidemyass.com/",
		UsernameLabel:   "Username",
		PasswordLabel:   "Password",
		Notes:           "Get VPN credentials from your HMA dashboard.",
	},
	"perfectprivacy": {
		Name:            "Perfect Privacy",
		ID:              "perfectprivacy",
		AuthType:        VPNAuthUserPass,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://www.perfect-privacy.com/en/member",
		UsernameLabel:   "Username",
		PasswordLabel:   "Password",
		Notes:           "Use your Perfect Privacy account credentials.",
	},
	"airvpn": {
		Name:            "AirVPN",
		ID:              "airvpn",
		AuthType:        VPNAuthToken,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://airvpn.org/devices/",
		TokenLabel:      "Device Key",
		Notes:           "Generate a device key from the AirVPN client area.",
	},
	"custom": {
		Name:            "Custom/OpenVPN",
		ID:              "custom",
		AuthType:        VPNAuthConfig,
		SupportsWG:      true,
		SupportsOpenVPN: true,
		CredDocsURL:     "https://github.com/qdm12/gluetun-wiki",
		UsernameLabel:   "Username (optional)",
		PasswordLabel:   "Password (optional)",
		Notes:           "Place your .ovpn file in configs/gluetun/ and configure manually.",
	},
}

// GetVPNProvider returns the provider info for a given provider ID
func GetVPNProvider(id string) (VPNProvider, bool) {
	provider, ok := VPNProviders[id]
	return provider, ok
}

// GetVPNProviderIDs returns a sorted list of provider IDs
func GetVPNProviderIDs() []string {
	return []string{
		"nordvpn", "mullvad", "protonvpn", "pia", "surfshark",
		"expressvpn", "windscribe", "ipvanish", "cyberghost", "ivpn",
		"torguard", "vyprvpn", "purevpn", "hidemyass", "perfectprivacy",
		"airvpn", "custom",
	}
}
