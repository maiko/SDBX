package handlers

import (
	"html/template"
	"net/http"

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

// HandleDashboard handles the main dashboard page
func (h *DashboardHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildDashboardData(r)
	if err != nil {
		httpError(w, "dashboard.buildData", err, http.StatusInternalServerError)
		return
	}
	h.renderTemplate(w, "pages/dashboard.html", data)
}

// HandleServicesGrid returns the services grid HTML fragment for htmx polling
func (h *DashboardHandler) HandleServicesGrid(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildDashboardData(r)
	if err != nil {
		httpError(w, "dashboard.servicesGrid", err, http.StatusInternalServerError)
		return
	}
	renderTemplate(h.templates, w, "service-grid-fragment", "dashboard.grid", data)
}

func (h *DashboardHandler) buildDashboardData(r *http.Request) (map[string]interface{}, error) {
	ctx := r.Context()

	serviceMap, err := buildServiceInfoMap(h.compose, h.registry, ctx)
	if err != nil {
		return nil, err
	}

	// Build quick access list (services with web UI)
	var quickAccess []ServiceInfo
	for _, svc := range serviceMap {
		if svc.HasWebUI && svc.URL != "" {
			quickAccess = append(quickAccess, svc)
		}
	}

	data := map[string]interface{}{
		"ServicesByCategory": groupByCategory(serviceMap),
		"TotalServices":      len(serviceMap),
		"RunningServices":    countRunningServices(serviceMap),
		"QuickAccess":        quickAccess,
	}
	return data, nil
}

func (h *DashboardHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	renderTemplate(h.templates, w, name, "dashboard", data)
}
