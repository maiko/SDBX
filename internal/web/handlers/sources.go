package handlers

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"time"

	"github.com/maiko/sdbx/internal/registry"
)

// validSourceName matches valid source names: starts with lowercase alphanumeric,
// followed by up to 63 lowercase alphanumeric, hyphen, or underscore characters.
var validSourceName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// validateSourceName checks whether a source name is valid.
func validateSourceName(name string) bool {
	return validSourceName.MatchString(name)
}

const (
	sourceUpdateTimeout    = 60 * time.Second
	sourceUpdateAllTimeout = 120 * time.Second
)

// SourcesHandler handles source management routes
type SourcesHandler struct {
	registry  *registry.Registry
	templates *template.Template
}

// NewSourcesHandler creates a new sources handler
func NewSourcesHandler(reg *registry.Registry, tmpl *template.Template) *SourcesHandler {
	return &SourcesHandler{
		registry:  reg,
		templates: tmpl,
	}
}

// SourceDisplay represents a source for display in templates
type SourceDisplay struct {
	Name        string
	Type        string // "embedded", "git", "local"
	URL         string
	Priority    int
	Enabled     bool
	Branch      string
	LastCommit  string
	LastUpdated string
}

// SourceResponse represents API response for source operations
type SourceResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Source  string `json:"source,omitempty"`
}

// HandleSourcesPage handles the source management page
func (h *SourcesHandler) HandleSourcesPage(w http.ResponseWriter, r *http.Request) {
	sources := h.registry.Sources()

	var displays []SourceDisplay
	for _, src := range sources {
		display := SourceDisplay{
			Name:     src.Name(),
			Type:     src.Type(),
			Priority: src.Priority(),
			Enabled:  src.IsEnabled(),
		}

		// Extract git-specific details
		if gitSrc, ok := src.(*registry.GitSource); ok {
			display.URL = gitSrc.GetURL()
			display.Branch = gitSrc.GetBranch()
			commit := gitSrc.GetCommit()
			if len(commit) > 12 {
				display.LastCommit = commit[:12]
			} else if commit != "" {
				display.LastCommit = commit
			}
			lastUpdated := gitSrc.GetLastUpdated()
			if !lastUpdated.IsZero() {
				display.LastUpdated = lastUpdated.Format("2006-01-02 15:04:05")
			}
		}

		displays = append(displays, display)
	}

	data := map[string]interface{}{
		"Sources":      displays,
		"TotalSources": len(displays),
	}

	h.renderTemplate(w, "pages/sources.html", data)
}

// HandleAddSource handles POST /api/sources/add
func (h *SourcesHandler) HandleAddSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondJSON(w, http.StatusMethodNotAllowed, SourceResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	name := r.FormValue("name")
	url := r.FormValue("url")
	branch := r.FormValue("branch")

	if name == "" || url == "" {
		h.respondJSON(w, http.StatusBadRequest, SourceResponse{
			Success: false,
			Message: "Name and URL are required",
		})
		return
	}

	if !validateSourceName(name) {
		h.respondJSON(w, http.StatusBadRequest, SourceResponse{
			Success: false,
			Message: "Invalid source name: must match ^[a-z0-9][a-z0-9_-]{0,63}$",
		})
		return
	}

	if branch == "" {
		branch = "main"
	}

	// Validate name doesn't conflict with reserved names
	if name == "embedded" {
		h.respondJSON(w, http.StatusBadRequest, SourceResponse{
			Success: false,
			Message: "Cannot use reserved name 'embedded'",
		})
		return
	}

	src := registry.Source{
		Name:     name,
		Type:     "git",
		URL:      url,
		Branch:   branch,
		Priority: 50,
		Enabled:  true,
	}

	if err := h.registry.AddSource(src); err != nil {
		jsonError(w, fmt.Sprintf("Failed to add source '%s'", name), "sources.Add", err, http.StatusConflict)
		return
	}

	h.respondJSON(w, http.StatusOK, SourceResponse{
		Success: true,
		Message: fmt.Sprintf("Source '%s' added successfully", name),
		Source:  name,
	})
}

// HandleRemoveSource handles POST /api/sources/{source}/remove
func (h *SourcesHandler) HandleRemoveSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondJSON(w, http.StatusMethodNotAllowed, SourceResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	sourceName := r.PathValue("source")
	if sourceName == "" {
		h.respondJSON(w, http.StatusBadRequest, SourceResponse{
			Success: false,
			Message: "Source name is required",
		})
		return
	}
	if !validateSourceName(sourceName) {
		h.respondJSON(w, http.StatusBadRequest, SourceResponse{
			Success: false,
			Message: "Invalid source name",
		})
		return
	}

	// Prevent removing the embedded source
	if sourceName == "embedded" {
		h.respondJSON(w, http.StatusBadRequest, SourceResponse{
			Success: false,
			Message: "Cannot remove the embedded source",
		})
		return
	}

	if err := h.registry.RemoveSource(sourceName); err != nil {
		jsonError(w, fmt.Sprintf("Failed to remove source '%s'", sourceName), "sources.Remove", err, http.StatusNotFound)
		return
	}

	h.respondJSON(w, http.StatusOK, SourceResponse{
		Success: true,
		Message: fmt.Sprintf("Source '%s' removed successfully", sourceName),
		Source:  sourceName,
	})
}

// HandleUpdateSource handles POST /api/sources/{source}/update
func (h *SourcesHandler) HandleUpdateSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondJSON(w, http.StatusMethodNotAllowed, SourceResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	sourceName := r.PathValue("source")
	if sourceName == "" {
		h.respondJSON(w, http.StatusBadRequest, SourceResponse{
			Success: false,
			Message: "Source name is required",
		})
		return
	}
	if !validateSourceName(sourceName) {
		h.respondJSON(w, http.StatusBadRequest, SourceResponse{
			Success: false,
			Message: "Invalid source name",
		})
		return
	}

	src, err := h.registry.GetSource(sourceName)
	if err != nil {
		jsonError(w, fmt.Sprintf("Source '%s' not found", sourceName), "sources.Update", err, http.StatusNotFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), sourceUpdateTimeout)
	defer cancel()

	if err := src.Update(ctx); err != nil {
		jsonError(w, fmt.Sprintf("Failed to update source '%s'", sourceName), "sources.Update", err, http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, http.StatusOK, SourceResponse{
		Success: true,
		Message: fmt.Sprintf("Source '%s' updated successfully", sourceName),
		Source:  sourceName,
	})
}

// HandleUpdateAllSources handles POST /api/sources/update-all
func (h *SourcesHandler) HandleUpdateAllSources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondJSON(w, http.StatusMethodNotAllowed, SourceResponse{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), sourceUpdateAllTimeout)
	defer cancel()

	if err := h.registry.Update(ctx); err != nil {
		jsonError(w, "Failed to update some sources", "sources.UpdateAll", err, http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, http.StatusOK, SourceResponse{
		Success: true,
		Message: "All sources updated successfully",
	})
}

func (h *SourcesHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	respondJSON(w, statusCode, data)
}

func (h *SourcesHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	renderTemplate(h.templates, w, name, "sources", data)
}
