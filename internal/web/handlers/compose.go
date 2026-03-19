package handlers

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
)

// ComposeHandler handles the compose file viewer route
type ComposeHandler struct {
	projectDir string
	templates  *template.Template
}

// NewComposeHandler creates a new compose handler
func NewComposeHandler(projectDir string, tmpl *template.Template) *ComposeHandler {
	return &ComposeHandler{
		projectDir: projectDir,
		templates:  tmpl,
	}
}

// HandleComposePage handles the compose file viewer page
func (h *ComposeHandler) HandleComposePage(w http.ResponseWriter, r *http.Request) {
	composePath := filepath.Join(h.projectDir, "compose.yaml")

	data := map[string]interface{}{
		"ComposeExists": false,
	}

	content, err := os.ReadFile(composePath)
	if err != nil {
		if !os.IsNotExist(err) {
			httpError(w, "compose.ReadFile", err, http.StatusInternalServerError)
			return
		}
	} else {
		data["ComposeExists"] = true
		data["ComposeContent"] = string(content)
	}

	h.renderTemplate(w, "pages/compose.html", data)
}

func (h *ComposeHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	renderTemplate(h.templates, w, name, "compose", data)
}
