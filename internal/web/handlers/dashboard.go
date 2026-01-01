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
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>SDBX Dashboard</title>
    <link rel="stylesheet" href="/static/css/colors.css">
    <link rel="stylesheet" href="/static/css/main.css">
</head>
<body>
    <div class="container">
        <h1>SDBX Dashboard</h1>
        <p>Service management dashboard will be available here soon...</p>
    </div>
</body>
</html>
	`))
}
