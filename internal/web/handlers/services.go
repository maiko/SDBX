package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/maiko/sdbx/internal/docker"
	"github.com/maiko/sdbx/internal/registry"
)

// ServicesHandler handles service management routes
type ServicesHandler struct {
	compose   *docker.Compose
	registry  *registry.Registry
	templates *template.Template
}

// NewServicesHandler creates a new services handler
func NewServicesHandler(compose *docker.Compose, reg *registry.Registry, tmpl *template.Template) *ServicesHandler {
	return &ServicesHandler{
		compose:   compose,
		registry:  reg,
		templates: tmpl,
	}
}

// ServiceResponse represents API response for service operations
type ServiceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Service string `json:"service,omitempty"`
	Status  string `json:"status,omitempty"`
}

// HandleServicesPage handles the services management page
func (h *ServicesHandler) HandleServicesPage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get service status from Docker
	dockerServices, err := h.compose.PS(ctx)
	if err != nil {
		dockerServices = []docker.Service{}
	}

	// Get service definitions from registry
	registryServices, err := h.registry.ListServices(ctx)
	if err != nil {
		http.Error(w, "Failed to load services", http.StatusInternalServerError)
		return
	}

	// Create service info map (same logic as dashboard)
	serviceMap := make(map[string]ServiceInfo)

	for _, regSvc := range registryServices {
		serviceMap[regSvc.Name] = ServiceInfo{
			Name:        regSvc.Name,
			DisplayName: formatServiceName(regSvc.Name),
			Status:      "unknown",
			Health:      "",
			Running:     false,
			Category:    string(regSvc.Category),
			Description: regSvc.Description,
			HasWebUI:    regSvc.HasWebUI,
			URL:         "",
		}
	}

	for _, dockerSvc := range dockerServices {
		serviceName := strings.TrimPrefix(dockerSvc.Name, "sdbx-")
		if info, exists := serviceMap[serviceName]; exists {
			info.Status = dockerSvc.Status
			info.Health = dockerSvc.Health
			info.Running = dockerSvc.Running
			serviceMap[serviceName] = info
		}
	}

	// Convert to slice
	var services []ServiceInfo
	for _, svc := range serviceMap {
		services = append(services, svc)
	}

	data := map[string]interface{}{
		"Services": services,
	}

	h.renderTemplate(w, "pages/services.html", data)
}

// HandleGetServices handles GET /api/services - returns service list as JSON
func (h *ServicesHandler) HandleGetServices(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Get service status from Docker
	dockerServices, err := h.compose.PS(ctx)
	if err != nil {
		h.respondJSON(w, http.StatusInternalServerError, ServiceResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get services: %v", err),
		})
		return
	}

	// Get service definitions from registry
	registryServices, err := h.registry.ListServices(ctx)
	if err != nil {
		h.respondJSON(w, http.StatusInternalServerError, ServiceResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to load service definitions: %v", err),
		})
		return
	}

	// Create service info map
	serviceMap := make(map[string]ServiceInfo)

	for _, regSvc := range registryServices {
		serviceMap[regSvc.Name] = ServiceInfo{
			Name:        regSvc.Name,
			DisplayName: formatServiceName(regSvc.Name),
			Status:      "unknown",
			Health:      "",
			Running:     false,
			Category:    string(regSvc.Category),
			Description: regSvc.Description,
			HasWebUI:    regSvc.HasWebUI,
			URL:         "",
		}
	}

	for _, dockerSvc := range dockerServices {
		serviceName := strings.TrimPrefix(dockerSvc.Name, "sdbx-")
		if info, exists := serviceMap[serviceName]; exists {
			info.Status = dockerSvc.Status
			info.Health = dockerSvc.Health
			info.Running = dockerSvc.Running
			serviceMap[serviceName] = info
		}
	}

	// Convert to slice
	var services []ServiceInfo
	for _, svc := range serviceMap {
		services = append(services, svc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

// HandleStartService handles POST /api/services/{service}/start
func (h *ServicesHandler) HandleStartService(w http.ResponseWriter, r *http.Request) {
	serviceName := r.PathValue("service")
	if serviceName == "" {
		h.respondJSON(w, http.StatusBadRequest, ServiceResponse{
			Success: false,
			Message: "Service name is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Start the service
	if err := h.compose.Start(ctx, serviceName); err != nil {
		h.respondJSON(w, http.StatusInternalServerError, ServiceResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to start service: %v", err),
			Service: serviceName,
		})
		return
	}

	h.respondJSON(w, http.StatusOK, ServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Service %s started successfully", serviceName),
		Service: serviceName,
		Status:  "starting",
	})
}

// HandleStopService handles POST /api/services/{service}/stop
func (h *ServicesHandler) HandleStopService(w http.ResponseWriter, r *http.Request) {
	serviceName := r.PathValue("service")
	if serviceName == "" {
		h.respondJSON(w, http.StatusBadRequest, ServiceResponse{
			Success: false,
			Message: "Service name is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Stop the service
	if err := h.compose.Stop(ctx, serviceName); err != nil {
		h.respondJSON(w, http.StatusInternalServerError, ServiceResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to stop service: %v", err),
			Service: serviceName,
		})
		return
	}

	h.respondJSON(w, http.StatusOK, ServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Service %s stopped successfully", serviceName),
		Service: serviceName,
		Status:  "stopped",
	})
}

// HandleRestartService handles POST /api/services/{service}/restart
func (h *ServicesHandler) HandleRestartService(w http.ResponseWriter, r *http.Request) {
	serviceName := r.PathValue("service")
	if serviceName == "" {
		h.respondJSON(w, http.StatusBadRequest, ServiceResponse{
			Success: false,
			Message: "Service name is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	// Restart the service
	if err := h.compose.Restart(ctx, serviceName); err != nil {
		h.respondJSON(w, http.StatusInternalServerError, ServiceResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to restart service: %v", err),
			Service: serviceName,
		})
		return
	}

	h.respondJSON(w, http.StatusOK, ServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Service %s restarted successfully", serviceName),
		Service: serviceName,
		Status:  "restarting",
	})
}

// respondJSON sends a JSON response
func (h *ServicesHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// renderTemplate renders a template with data
func (h *ServicesHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		httpError(w, "services template render", err, http.StatusInternalServerError)
	}
}
