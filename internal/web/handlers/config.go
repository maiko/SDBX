package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/maiko/sdbx/internal/config"
)

// ConfigHandler handles configuration editing routes
type ConfigHandler struct {
	projectDir string
	templates  *template.Template
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(projectDir string, tmpl *template.Template) *ConfigHandler {
	return &ConfigHandler{
		projectDir: projectDir,
		templates:  tmpl,
	}
}

// ConfigResponse represents API response for config operations
type ConfigResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
}

// HandleConfigPage handles the config editor page
func (h *ConfigHandler) HandleConfigPage(w http.ResponseWriter, r *http.Request) {
	configPath := filepath.Join(h.projectDir, ".sdbx.yaml")

	// Read current config
	content, err := os.ReadFile(configPath)
	if err != nil {
		http.Error(w, "Failed to read config file", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"ConfigContent": string(content),
		"ConfigPath":    ".sdbx.yaml",
	}

	h.renderTemplate(w, "pages/config.html", data)
}

// HandleGetConfig handles GET /api/config
func (h *ConfigHandler) HandleGetConfig(w http.ResponseWriter, r *http.Request) {
	configPath := filepath.Join(h.projectDir, ".sdbx.yaml")

	content, err := os.ReadFile(configPath)
	if err != nil {
		h.respondJSON(w, http.StatusInternalServerError, ConfigResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to read config: %v", err),
		})
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

// HandleValidateConfig handles POST /api/config/validate
func (h *ConfigHandler) HandleValidateConfig(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.respondJSON(w, http.StatusBadRequest, ConfigResponse{
			Success: false,
			Message: "Invalid form data",
		})
		return
	}

	content := r.FormValue("content")
	if content == "" {
		h.respondJSON(w, http.StatusBadRequest, ConfigResponse{
			Success: false,
			Message: "Config content is required",
		})
		return
	}

	// Validate YAML syntax
	var testConfig map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &testConfig); err != nil {
		h.respondJSON(w, http.StatusBadRequest, ConfigResponse{
			Success: false,
			Message: "Invalid YAML syntax",
			Errors:  []string{err.Error()},
		})
		return
	}

	// Try to parse as Config struct for stricter validation
	var cfg config.Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		h.respondJSON(w, http.StatusBadRequest, ConfigResponse{
			Success: false,
			Message: "Config validation failed",
			Errors:  []string{err.Error()},
		})
		return
	}

	// Additional validation
	errors := h.validateConfig(&cfg)
	if len(errors) > 0 {
		h.respondJSON(w, http.StatusBadRequest, ConfigResponse{
			Success: false,
			Message: "Config validation failed",
			Errors:  errors,
		})
		return
	}

	h.respondJSON(w, http.StatusOK, ConfigResponse{
		Success: true,
		Message: "Config is valid",
	})
}

// HandleSaveConfig handles POST /api/config/save
func (h *ConfigHandler) HandleSaveConfig(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.respondJSON(w, http.StatusBadRequest, ConfigResponse{
			Success: false,
			Message: "Invalid form data",
		})
		return
	}

	content := r.FormValue("content")
	if content == "" {
		h.respondJSON(w, http.StatusBadRequest, ConfigResponse{
			Success: false,
			Message: "Config content is required",
		})
		return
	}

	// Validate before saving
	var cfg config.Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		h.respondJSON(w, http.StatusBadRequest, ConfigResponse{
			Success: false,
			Message: "Invalid YAML syntax",
			Errors:  []string{err.Error()},
		})
		return
	}

	// Additional validation
	errors := h.validateConfig(&cfg)
	if len(errors) > 0 {
		h.respondJSON(w, http.StatusBadRequest, ConfigResponse{
			Success: false,
			Message: "Config validation failed",
			Errors:  errors,
		})
		return
	}

	// Backup existing config
	configPath := filepath.Join(h.projectDir, ".sdbx.yaml")
	backupPath := configPath + ".backup"

	if _, err := os.Stat(configPath); err == nil {
		if err := os.Rename(configPath, backupPath); err != nil {
			h.respondJSON(w, http.StatusInternalServerError, ConfigResponse{
				Success: false,
				Message: fmt.Sprintf("Failed to backup config: %v", err),
			})
			return
		}
	}

	// Write new config
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		// Try to restore backup
		os.Rename(backupPath, configPath)

		h.respondJSON(w, http.StatusInternalServerError, ConfigResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to save config: %v", err),
		})
		return
	}

	// Remove backup on success
	os.Remove(backupPath)

	h.respondJSON(w, http.StatusOK, ConfigResponse{
		Success: true,
		Message: "Config saved successfully. Run 'sdbx down && sdbx up' to apply changes.",
	})
}

// validateConfig performs additional validation on config
func (h *ConfigHandler) validateConfig(cfg *config.Config) []string {
	var errors []string

	// Domain validation
	if cfg.Domain == "" {
		errors = append(errors, "domain is required")
	}

	// Timezone validation
	if cfg.Timezone == "" {
		errors = append(errors, "timezone is required")
	}

	// Expose mode validation
	validExposeModes := []string{"lan", "direct", "cloudflared"}
	isValidExposeMode := false
	for _, mode := range validExposeModes {
		if cfg.Expose.Mode == mode {
			isValidExposeMode = true
			break
		}
	}
	if !isValidExposeMode {
		errors = append(errors, fmt.Sprintf("expose.mode must be one of: %v", validExposeModes))
	}

	// Routing strategy validation
	validStrategies := []string{"subdomain", "path"}
	isValidStrategy := false
	for _, strategy := range validStrategies {
		if cfg.Routing.Strategy == strategy {
			isValidStrategy = true
			break
		}
	}
	if !isValidStrategy {
		errors = append(errors, fmt.Sprintf("routing.strategy must be one of: %v", validStrategies))
	}

	// Path validation
	if cfg.MediaPath == "" {
		errors = append(errors, "media_path is required")
	}
	if cfg.DownloadsPath == "" {
		errors = append(errors, "downloads_path is required")
	}
	if cfg.ConfigPath == "" {
		errors = append(errors, "config_path is required")
	}

	return errors
}

// respondJSON sends a JSON response
func (h *ConfigHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// renderTemplate renders a template with data
func (h *ConfigHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		httpError(w, "config template render", err, http.StatusInternalServerError)
	}
}
