package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/maiko/sdbx/internal/config"
)

// VPNHandler handles VPN configuration routes
type VPNHandler struct {
	projectDir string
	templates  *template.Template
}

// NewVPNHandler creates a new VPN handler
func NewVPNHandler(projectDir string, tmpl *template.Template) *VPNHandler {
	return &VPNHandler{
		projectDir: projectDir,
		templates:  tmpl,
	}
}

// VPNResponse represents API response for VPN operations
type VPNResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ProviderJSON represents a VPN provider for JSON API responses
type ProviderJSON struct {
	Name            string `json:"name"`
	ID              string `json:"id"`
	AuthType        string `json:"authType"`
	SupportsWG      bool   `json:"supportsWG"`
	SupportsOpenVPN bool   `json:"supportsOpenVPN"`
	CredDocsURL     string `json:"credDocsURL"`
	UsernameLabel   string `json:"usernameLabel,omitempty"`
	PasswordLabel   string `json:"passwordLabel,omitempty"`
	TokenLabel      string `json:"tokenLabel,omitempty"`
	Notes           string `json:"notes,omitempty"`
}

// HandleVPNPage handles the VPN configuration page
func (h *VPNHandler) HandleVPNPage(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Build provider list in sorted order
	providerIDs := config.GetVPNProviderIDs()
	var providers []config.VPNProvider
	for _, id := range providerIDs {
		if p, ok := config.GetVPNProvider(id); ok {
			providers = append(providers, p)
		}
	}

	data := map[string]interface{}{
		"VPNEnabled":      cfg.VPNEnabled,
		"CurrentProvider": cfg.VPNProvider,
		"CurrentType":     cfg.VPNType,
		"CurrentCountry":  cfg.VPNCountry,
		"Providers":       providers,
	}

	h.renderTemplate(w, "pages/vpn.html", data)
}

// HandleVPNProviders handles GET /api/vpn/providers - returns provider list as JSON
func (h *VPNHandler) HandleVPNProviders(w http.ResponseWriter, r *http.Request) {
	providerIDs := config.GetVPNProviderIDs()
	var providers []ProviderJSON

	for _, id := range providerIDs {
		p, ok := config.GetVPNProvider(id)
		if !ok {
			continue
		}
		providers = append(providers, ProviderJSON{
			Name:            p.Name,
			ID:              id,
			AuthType:        string(p.AuthType),
			SupportsWG:      p.SupportsWG,
			SupportsOpenVPN: p.SupportsOpenVPN,
			CredDocsURL:     p.CredDocsURL,
			UsernameLabel:   p.UsernameLabel,
			PasswordLabel:   p.PasswordLabel,
			TokenLabel:      p.TokenLabel,
			Notes:           p.Notes,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

// HandleVPNConfigure handles POST /api/vpn/configure - updates VPN config
func (h *VPNHandler) HandleVPNConfigure(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.respondJSON(w, http.StatusBadRequest, VPNResponse{
			Success: false,
			Message: "Invalid form data",
		})
		return
	}

	cfg, err := config.Load()
	if err != nil {
		jsonError(w, "Failed to load configuration", "vpn.Configure.Load", err, http.StatusInternalServerError)
		return
	}

	// Read form values
	enabledStr := r.FormValue("vpn_enabled")
	provider := r.FormValue("vpn_provider")
	vpnType := r.FormValue("vpn_type")
	country := r.FormValue("vpn_country")

	// Update config
	cfg.VPNEnabled = enabledStr == "true" || enabledStr == "on"
	cfg.VPNProvider = provider
	cfg.VPNCountry = country

	// Validate VPN type
	if vpnType != "" {
		if vpnType != "wireguard" && vpnType != "openvpn" {
			h.respondJSON(w, http.StatusBadRequest, VPNResponse{
				Success: false,
				Message: "VPN type must be 'wireguard' or 'openvpn'",
			})
			return
		}
		cfg.VPNType = vpnType
	}

	// Validate: if VPN is enabled, provider is required
	if cfg.VPNEnabled && cfg.VPNProvider == "" {
		h.respondJSON(w, http.StatusBadRequest, VPNResponse{
			Success: false,
			Message: "VPN provider is required when VPN is enabled",
		})
		return
	}

	// Validate provider exists if specified
	if cfg.VPNProvider != "" {
		if _, ok := config.GetVPNProvider(cfg.VPNProvider); !ok {
			h.respondJSON(w, http.StatusBadRequest, VPNResponse{
				Success: false,
				Message: "Unknown VPN provider",
			})
			return
		}
	}

	// Save config
	configPath := filepath.Join(h.projectDir, ".sdbx.yaml")
	if err := cfg.Save(configPath); err != nil {
		jsonError(w, "Failed to save VPN configuration", "vpn.Configure.Save", err, http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, http.StatusOK, VPNResponse{
		Success: true,
		Message: "VPN configuration saved. Run 'sdbx down && sdbx up' to apply changes.",
	})
}

func (h *VPNHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	respondJSON(w, statusCode, data)
}

func (h *VPNHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	renderTemplate(h.templates, w, name, "vpn", data)
}
