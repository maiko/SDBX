package handlers

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"sort"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/registry"
)

// ServiceInfoHandler handles service information display
type ServiceInfoHandler struct {
	registry  *registry.Registry
	templates *template.Template
}

// NewServiceInfoHandler creates a new service info handler
func NewServiceInfoHandler(reg *registry.Registry, tmpl *template.Template) *ServiceInfoHandler {
	return &ServiceInfoHandler{
		registry:  reg,
		templates: tmpl,
	}
}

// ServiceConnectionInfo represents service connection information for display
type ServiceConnectionInfo struct {
	Name         string
	Description  string
	Category     string
	DockerHost   string
	InternalPort int
	ExternalURL  string
	HasWebUI     bool
}

// HandleServiceInfoPage displays service connection information
func (h *ServiceInfoHandler) HandleServiceInfoPage(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		httpError(w, "load config", err, http.StatusInternalServerError)
		return
	}

	// Get all services
	services, err := h.registry.ListServices(ctx)
	if err != nil {
		httpError(w, "list services", err, http.StatusInternalServerError)
		return
	}

	// Build service info list and get full definitions for port info
	serviceInfos := make([]ServiceConnectionInfo, 0, len(services))
	for _, svc := range services {
		info := ServiceConnectionInfo{
			Name:        svc.Name,
			Description: svc.Description,
			Category:    string(svc.Category),
			DockerHost:  fmt.Sprintf("sdbx-%s", svc.Name),
			HasWebUI:    svc.HasWebUI,
		}

		// Get full service definition for port info
		if svcDef, _, err := h.registry.GetService(ctx, svc.Name); err == nil {
			if svcDef.Routing.Port > 0 {
				info.InternalPort = svcDef.Routing.Port
			}
		}

		// Get external URL if has web UI
		if svc.HasWebUI {
			info.ExternalURL = cfg.GetServiceURL(svc.Name)
		}

		serviceInfos = append(serviceInfos, info)
	}

	// Sort by category then name
	sort.Slice(serviceInfos, func(i, j int) bool {
		if serviceInfos[i].Category == serviceInfos[j].Category {
			return serviceInfos[i].Name < serviceInfos[j].Name
		}
		return serviceInfos[i].Category < serviceInfos[j].Category
	})

	data := map[string]interface{}{
		"Services": serviceInfos,
	}

	if err := h.templates.ExecuteTemplate(w, "pages/service_info.html", data); err != nil {
		httpError(w, "render template", err, http.StatusInternalServerError)
	}
}
