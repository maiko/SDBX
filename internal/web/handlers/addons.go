package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/registry"
)

// AddonsHandler handles addon management routes
type AddonsHandler struct {
	registry   *registry.Registry
	projectDir string
	templates  *template.Template
}

// NewAddonsHandler creates a new addons handler
func NewAddonsHandler(reg *registry.Registry, projectDir string, tmpl *template.Template) *AddonsHandler {
	return &AddonsHandler{
		registry:   reg,
		projectDir: projectDir,
		templates:  tmpl,
	}
}

// AddonDisplay represents an addon for display
type AddonDisplay struct {
	Name        string
	DisplayName string
	Description string
	Category    string
	Version     string
	Source      string
	Enabled     bool
	HasWebUI    bool
}

// AddonResponse represents API response for addon operations
type AddonResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Addon   string `json:"addon,omitempty"`
}

// HandleAddonsPage handles the addons catalog page
func (h *AddonsHandler) HandleAddonsPage(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Load config to check enabled addons
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Get all services from registry
	services, err := h.registry.ListServices(ctx)
	if err != nil {
		http.Error(w, "Failed to load addons", http.StatusInternalServerError)
		return
	}

	// Filter and format addons
	var addons []AddonDisplay
	for _, svc := range services {
		if svc.IsAddon {
			addons = append(addons, AddonDisplay{
				Name:        svc.Name,
				DisplayName: formatServiceName(svc.Name),
				Description: svc.Description,
				Category:    string(svc.Category),
				Version:     svc.Version,
				Source:      svc.Source,
				Enabled:     cfg.IsAddonEnabled(svc.Name),
				HasWebUI:    svc.HasWebUI,
			})
		}
	}

	// Group by category
	addonsByCategory := make(map[string][]AddonDisplay)
	for _, addon := range addons {
		category := addon.Category
		if category == "" {
			category = "other"
		}
		addonsByCategory[category] = append(addonsByCategory[category], addon)
	}

	data := map[string]interface{}{
		"AddonsByCategory": addonsByCategory,
		"TotalAddons":      len(addons),
		"EnabledAddons":    countEnabledAddons(addons),
	}

	h.renderTemplate(w, "pages/addons.html", data)
}

// HandleSearchAddons handles addon search
func (h *AddonsHandler) HandleSearchAddons(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	category := r.URL.Query().Get("category")

	ctx := context.Background()

	var categoryFilter registry.ServiceCategory
	if category != "" {
		categoryFilter = registry.ServiceCategory(category)
	}

	// Search in registry
	results, err := h.registry.SearchServices(ctx, query, categoryFilter)
	if err != nil {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	// Load config to check enabled status
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Filter to addons only
	var addons []AddonDisplay
	for _, svc := range results {
		if svc.IsAddon {
			addons = append(addons, AddonDisplay{
				Name:        svc.Name,
				DisplayName: formatServiceName(svc.Name),
				Description: svc.Description,
				Category:    string(svc.Category),
				Version:     svc.Version,
				Source:      svc.Source,
				Enabled:     cfg.IsAddonEnabled(svc.Name),
				HasWebUI:    svc.HasWebUI,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(addons)
}

// HandleEnableAddon handles POST /api/addons/{addon}/enable
func (h *AddonsHandler) HandleEnableAddon(w http.ResponseWriter, r *http.Request) {
	addonName := r.PathValue("addon")
	if addonName == "" {
		h.respondJSON(w, http.StatusBadRequest, AddonResponse{
			Success: false,
			Message: "Addon name is required",
		})
		return
	}

	ctx := context.Background()

	// Validate addon exists in registry
	def, _, err := h.registry.GetService(ctx, addonName)
	if err != nil {
		h.respondJSON(w, http.StatusNotFound, AddonResponse{
			Success: false,
			Message: fmt.Sprintf("Addon '%s' not found", addonName),
			Addon:   addonName,
		})
		return
	}

	// Check if it's actually an addon
	if !def.Conditions.RequireAddon {
		h.respondJSON(w, http.StatusBadRequest, AddonResponse{
			Success: false,
			Message: fmt.Sprintf("'%s' is a core service, not an addon", addonName),
			Addon:   addonName,
		})
		return
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Check if already enabled
	if cfg.IsAddonEnabled(addonName) {
		h.respondJSON(w, http.StatusOK, AddonResponse{
			Success: true,
			Message: fmt.Sprintf("Addon '%s' is already enabled", addonName),
			Addon:   addonName,
		})
		return
	}

	// Enable addon
	cfg.EnableAddon(addonName)

	// Save config
	if err := cfg.Save(".sdbx.yaml"); err != nil {
		h.respondJSON(w, http.StatusInternalServerError, AddonResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to save config: %v", err),
			Addon:   addonName,
		})
		return
	}

	h.respondJSON(w, http.StatusOK, AddonResponse{
		Success: true,
		Message: fmt.Sprintf("Enabled '%s'. Run 'sdbx up' to start the service.", addonName),
		Addon:   addonName,
	})
}

// HandleDisableAddon handles POST /api/addons/{addon}/disable
func (h *AddonsHandler) HandleDisableAddon(w http.ResponseWriter, r *http.Request) {
	addonName := r.PathValue("addon")
	if addonName == "" {
		h.respondJSON(w, http.StatusBadRequest, AddonResponse{
			Success: false,
			Message: "Addon name is required",
		})
		return
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Check if enabled
	if !cfg.IsAddonEnabled(addonName) {
		h.respondJSON(w, http.StatusOK, AddonResponse{
			Success: true,
			Message: fmt.Sprintf("Addon '%s' is not enabled", addonName),
			Addon:   addonName,
		})
		return
	}

	// Disable addon
	cfg.DisableAddon(addonName)

	// Save config
	if err := cfg.Save(".sdbx.yaml"); err != nil {
		h.respondJSON(w, http.StatusInternalServerError, AddonResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to save config: %v", err),
			Addon:   addonName,
		})
		return
	}

	h.respondJSON(w, http.StatusOK, AddonResponse{
		Success: true,
		Message: fmt.Sprintf("Disabled '%s'. Run 'sdbx down && sdbx up' to apply changes.", addonName),
		Addon:   addonName,
	})
}

// countEnabledAddons counts how many addons are enabled
func countEnabledAddons(addons []AddonDisplay) int {
	count := 0
	for _, addon := range addons {
		if addon.Enabled {
			count++
		}
	}
	return count
}

// respondJSON sends a JSON response
func (h *AddonsHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// renderTemplate renders a template with data
func (h *AddonsHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
