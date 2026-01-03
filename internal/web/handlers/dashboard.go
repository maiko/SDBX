package handlers

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/maiko/sdbx/internal/docker"
	"github.com/maiko/sdbx/internal/registry"
)

// DashboardHandler handles dashboard routes
type DashboardHandler struct {
	compose   *docker.Compose
	registry  *registry.Registry
	templates *template.Template
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(compose *docker.Compose, reg *registry.Registry, tmpl *template.Template) *DashboardHandler {
	return &DashboardHandler{
		compose:   compose,
		registry:  reg,
		templates: tmpl,
	}
}

// ServiceInfo represents service information for display
type ServiceInfo struct {
	Name        string
	DisplayName string
	Status      string
	Health      string
	Running     bool
	Category    string
	Description string
	URL         string
	HasWebUI    bool
}

// HandleDashboard handles the main dashboard page
func (h *DashboardHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get service status from Docker
	dockerServices, err := h.compose.PS(ctx)
	if err != nil {
		// If compose fails, show empty dashboard
		dockerServices = []docker.Service{}
	}

	// Get service definitions from registry
	registryServices, err := h.registry.ListServices(ctx)
	if err != nil {
		http.Error(w, "Failed to load services", http.StatusInternalServerError)
		return
	}

	// Create service info map
	serviceMap := make(map[string]ServiceInfo)

	// First, populate from registry (to get metadata)
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
			URL:         "", // Will be set based on config
		}
	}

	// Update with Docker status
	for _, dockerSvc := range dockerServices {
		// Extract service name from container name (sdbx-servicename)
		serviceName := strings.TrimPrefix(dockerSvc.Name, "sdbx-")

		if info, exists := serviceMap[serviceName]; exists {
			info.Status = dockerSvc.Status
			info.Health = dockerSvc.Health
			info.Running = dockerSvc.Running
			serviceMap[serviceName] = info
		}
	}

	// Convert map to slice and group by category
	servicesByCategory := make(map[string][]ServiceInfo)
	for _, svc := range serviceMap {
		category := svc.Category
		if category == "" {
			category = "other"
		}
		servicesByCategory[category] = append(servicesByCategory[category], svc)
	}

	data := map[string]interface{}{
		"ServicesByCategory": servicesByCategory,
		"TotalServices":      len(serviceMap),
		"RunningServices":    countRunningServices(serviceMap),
	}

	h.renderTemplate(w, "pages/dashboard.html", data)
}

// countRunningServices counts how many services are running
func countRunningServices(services map[string]ServiceInfo) int {
	count := 0
	for _, svc := range services {
		if svc.Running {
			count++
		}
	}
	return count
}

// renderTemplate renders a template with data
func (h *DashboardHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
