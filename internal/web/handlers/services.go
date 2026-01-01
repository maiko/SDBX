package handlers

import (
	"html/template"
	"net/http"

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

// HandleServicesPage handles the services management page
func (h *ServicesHandler) HandleServicesPage(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

// HandleGetServices handles GET /api/services
func (h *ServicesHandler) HandleGetServices(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

// HandleStartService handles POST /api/services/{service}/start
func (h *ServicesHandler) HandleStartService(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

// HandleStopService handles POST /api/services/{service}/stop
func (h *ServicesHandler) HandleStopService(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

// HandleRestartService handles POST /api/services/{service}/restart
func (h *ServicesHandler) HandleRestartService(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}
