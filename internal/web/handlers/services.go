package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/maiko/sdbx/internal/docker"
	"github.com/maiko/sdbx/internal/registry"
)

const (
	serviceQueryTimeout   = 10 * time.Second
	serviceStartTimeout   = 30 * time.Second
	serviceStopTimeout    = 30 * time.Second
	serviceRestartTimeout = 60 * time.Second
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

	serviceMap, err := buildServiceInfoMap(h.compose, h.registry, ctx)
	if err != nil {
		httpError(w, "services.buildInfoMap", err, http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"ServicesByCategory": groupByCategory(serviceMap),
	}

	h.renderTemplate(w, "pages/services.html", data)
}

// HandleGetServices handles GET /api/services - returns service list as JSON
func (h *ServicesHandler) HandleGetServices(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), serviceQueryTimeout)
	defer cancel()

	serviceMap, err := buildServiceInfoMap(h.compose, h.registry, ctx)
	if err != nil {
		jsonError(w, "Failed to load services", "services.List", err, http.StatusInternalServerError)
		return
	}

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

	ctx, cancel := context.WithTimeout(r.Context(), serviceStartTimeout)
	defer cancel()

	if err := h.compose.Start(ctx, serviceName); err != nil {
		jsonError(w, "Failed to start service", "services.Start", err, http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		h.renderServiceCard(w, r, serviceName, true)
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

	ctx, cancel := context.WithTimeout(r.Context(), serviceStopTimeout)
	defer cancel()

	if err := h.compose.Stop(ctx, serviceName); err != nil {
		jsonError(w, "Failed to stop service", "services.Stop", err, http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		h.renderServiceCard(w, r, serviceName, false)
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

	ctx, cancel := context.WithTimeout(r.Context(), serviceRestartTimeout)
	defer cancel()

	if err := h.compose.Restart(ctx, serviceName); err != nil {
		jsonError(w, "Failed to restart service", "services.Restart", err, http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		h.renderServiceCard(w, r, serviceName, true)
		return
	}

	h.respondJSON(w, http.StatusOK, ServiceResponse{
		Success: true,
		Message: fmt.Sprintf("Service %s restarted successfully", serviceName),
		Service: serviceName,
		Status:  "restarting",
	})
}

// renderServiceCard renders a service-card template fragment for htmx responses
func (h *ServicesHandler) renderServiceCard(w http.ResponseWriter, r *http.Request, serviceName string, running bool) {
	ctx := r.Context()

	info := ServiceInfo{
		Name:        serviceName,
		DisplayName: formatServiceName(serviceName),
		Running:     running,
		Status:      "stopped",
	}
	if running {
		info.Status = "running"
	}

	// Enrich with registry metadata
	if svcInfo, _, err := h.registry.GetService(ctx, serviceName); err == nil {
		info.Category = string(svcInfo.Metadata.Category)
		info.Description = svcInfo.Metadata.Description
		info.HasWebUI = svcInfo.Routing.Enabled
	}

	renderTemplate(h.templates, w, "service-card", "services.card", info)
}

func (h *ServicesHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	respondJSON(w, statusCode, data)
}

func (h *ServicesHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	renderTemplate(h.templates, w, name, "services", data)
}
