package handlers

import (
	"html/template"
	"net/http"
	"sort"

	"github.com/maiko/sdbx/internal/config"
	"github.com/maiko/sdbx/internal/registry"
)

// LockHandler handles lock file viewer routes
type LockHandler struct {
	registry   *registry.Registry
	projectDir string
	templates  *template.Template
}

// NewLockHandler creates a new lock handler
func NewLockHandler(reg *registry.Registry, projectDir string, tmpl *template.Template) *LockHandler {
	return &LockHandler{
		registry:   reg,
		projectDir: projectDir,
		templates:  tmpl,
	}
}

// LockedServiceInfo represents a locked service for template display
type LockedServiceInfo struct {
	Name    string
	Version string
	Image   string
	Source  string
	Enabled bool
}

// LockVerifyResponse represents the JSON response for lock verification
type LockVerifyResponse struct {
	Success bool                            `json:"success"`
	Message string                          `json:"message"`
	Results []registry.LockVerificationResult `json:"results,omitempty"`
}

// HandleLockPage handles the lock file viewer page
func (h *LockHandler) HandleLockPage(w http.ResponseWriter, r *http.Request) {
	exists := registry.LockFileExists(h.projectDir)

	data := map[string]interface{}{
		"LockFileExists": exists,
	}

	if exists {
		lockPath := registry.GetLockFilePath(h.projectDir)
		loader := registry.NewLoader()
		lockFile, err := loader.LoadLockFile(lockPath)
		if err != nil {
			httpError(w, "lock.LoadLockFile", err, http.StatusInternalServerError)
			return
		}

		var services []LockedServiceInfo
		for name, svc := range lockFile.Services {
			services = append(services, LockedServiceInfo{
				Name:    name,
				Version: svc.DefinitionVersion,
				Image:   svc.Image.Repository + ":" + svc.Image.Tag,
				Source:  svc.Source,
				Enabled: svc.Enabled,
			})
		}

		sort.Slice(services, func(i, j int) bool {
			return services[i].Name < services[j].Name
		})

		data["Services"] = services
		data["ServiceCount"] = len(services)
		data["CLIVersion"] = lockFile.Metadata.CLIVersion
		data["GeneratedAt"] = lockFile.Metadata.GeneratedAt.Format("2006-01-02 15:04:05 UTC")
		data["ConfigHash"] = lockFile.Metadata.ConfigHash
	}

	h.renderTemplate(w, "pages/lock.html", data)
}

// HandleLockVerify handles POST /api/lock/verify
func (h *LockHandler) HandleLockVerify(w http.ResponseWriter, r *http.Request) {
	if !registry.LockFileExists(h.projectDir) {
		h.respondJSON(w, http.StatusOK, LockVerifyResponse{
			Success: false,
			Message: "No lock file found",
		})
		return
	}

	lockPath := registry.GetLockFilePath(h.projectDir)
	loader := registry.NewLoader()
	lockFile, err := loader.LoadLockFile(lockPath)
	if err != nil {
		jsonError(w, "Failed to load lock file", "lock.LoadLockFile", err, http.StatusInternalServerError)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	lockManager := registry.NewLockManager(h.registry, "")
	results, err := lockManager.Verify(r.Context(), cfg, lockFile)
	if err != nil {
		jsonError(w, "Verification failed", "lock.Verify", err, http.StatusInternalServerError)
		return
	}

	if len(results) == 0 {
		h.respondJSON(w, http.StatusOK, LockVerifyResponse{
			Success: true,
			Message: "Lock file is up to date. No changes detected.",
		})
		return
	}

	respondJSON(w, http.StatusOK, LockVerifyResponse{
		Success: false,
		Message: "Lock file has differences from current state",
		Results: results,
	})
}

func (h *LockHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	respondJSON(w, statusCode, data)
}

func (h *LockHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	renderTemplate(h.templates, w, name, "lock", data)
}
