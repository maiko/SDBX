package handlers

import (
	"html/template"
	"net/http"

	"github.com/maiko/sdbx/internal/registry"
)

// SetupHandler handles setup wizard routes
type SetupHandler struct {
	registry   *registry.Registry
	projectDir string
	templates  *template.Template
}

// NewSetupHandler creates a new setup handler
func NewSetupHandler(reg *registry.Registry, projectDir string, tmpl *template.Template) *SetupHandler {
	return &SetupHandler{
		registry:   reg,
		projectDir: projectDir,
		templates:  tmpl,
	}
}

// HandleWelcome handles the welcome page
func (h *SetupHandler) HandleWelcome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>SDBX Setup</title>
    <link rel="stylesheet" href="/static/css/colors.css">
    <link rel="stylesheet" href="/static/css/main.css">
</head>
<body>
    <div class="container">
        <h1>Welcome to SDBX Setup</h1>
        <p>Configure your media automation stack</p>
        <p>Setup wizard will be available here soon...</p>
    </div>
</body>
</html>
	`))
}

// Stub handlers for other setup steps
func (h *SetupHandler) HandleDomain(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *SetupHandler) HandleAdmin(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *SetupHandler) HandleStorage(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *SetupHandler) HandleVPN(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *SetupHandler) HandleAddons(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *SetupHandler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}

func (h *SetupHandler) HandleComplete(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented yet", http.StatusNotImplemented)
}
