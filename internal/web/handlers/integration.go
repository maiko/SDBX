package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/maiko/sdbx/internal/backup"
	"github.com/maiko/sdbx/internal/integrate"
)

// IntegrationHandler handles integration and backup routes
type IntegrationHandler struct {
	projectDir string
	templates  *template.Template
}

// NewIntegrationHandler creates a new integration handler
func NewIntegrationHandler(projectDir string, tmpl *template.Template) *IntegrationHandler {
	return &IntegrationHandler{
		projectDir: projectDir,
		templates:  tmpl,
	}
}

// IntegrationResponse represents API response for integration operations
type IntegrationResponse struct {
	Success      bool                       `json:"success"`
	Message      string                     `json:"message"`
	Total        int                        `json:"total,omitempty"`
	Successful   int                        `json:"successful,omitempty"`
	Failed       int                        `json:"failed,omitempty"`
	Integrations []IntegrationResultDisplay `json:"integrations,omitempty"`
}

// IntegrationResultDisplay represents an integration result for display
type IntegrationResultDisplay struct {
	Service string `json:"service"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// BackupDisplay represents a backup for display
type BackupDisplay struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	SizeHuman string    `json:"sizeHuman"`
	Timestamp time.Time `json:"timestamp"`
	Age       string    `json:"age"`
	Hostname  string    `json:"hostname"`
}

// BackupResponse represents API response for backup operations
type BackupResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Backup  *BackupDisplay  `json:"backup,omitempty"`
	Backups []BackupDisplay `json:"backups,omitempty"`
}

// HandleIntegrationPage handles the integration center page
func (h *IntegrationHandler) HandleIntegrationPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"ProjectDir": h.projectDir,
	}

	h.renderTemplate(w, "pages/integration.html", data)
}

// HandleRunIntegration handles POST /api/integration/run
func (h *IntegrationHandler) HandleRunIntegration(w http.ResponseWriter, r *http.Request) {
	// Load service configurations
	services, err := integrate.LoadServicesFromConfig(h.projectDir)
	if err != nil {
		h.respondJSON(w, http.StatusInternalServerError, IntegrationResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to load service configurations: %v", err),
		})
		return
	}

	if len(services) == 0 {
		h.respondJSON(w, http.StatusOK, IntegrationResponse{
			Success: false,
			Message: "No services found to integrate. Make sure services are enabled and running.",
		})
		return
	}

	// Create integration config
	cfg := integrate.DefaultConfig()
	cfg.Services = services
	cfg.DryRun = false
	cfg.Verbose = false

	// Create integrator
	integrator := integrate.NewIntegrator(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Run integrations
	results, err := integrator.Run(ctx)
	if err != nil {
		h.respondJSON(w, http.StatusInternalServerError, IntegrationResponse{
			Success: false,
			Message: fmt.Sprintf("Integration failed: %v", err),
		})
		return
	}

	// Convert results for display
	displayResults := make([]IntegrationResultDisplay, 0, len(results))
	successCount := 0
	for _, r := range results {
		if r.Success {
			successCount++
		}
		display := IntegrationResultDisplay{
			Service: r.Service,
			Success: r.Success,
			Message: r.Message,
		}
		if r.Error != nil {
			display.Error = r.Error.Error()
		}
		displayResults = append(displayResults, display)
	}

	h.respondJSON(w, http.StatusOK, IntegrationResponse{
		Success:      successCount == len(results),
		Message:      fmt.Sprintf("Completed %d integrations (%d succeeded, %d failed)", len(results), successCount, len(results)-successCount),
		Total:        len(results),
		Successful:   successCount,
		Failed:       len(results) - successCount,
		Integrations: displayResults,
	})
}

// HandleBackupPage handles the backup management page
func (h *IntegrationHandler) HandleBackupPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"ProjectDir": h.projectDir,
	}

	h.renderTemplate(w, "pages/backup.html", data)
}

// HandleListBackups handles GET /api/backup/list
func (h *IntegrationHandler) HandleListBackups(w http.ResponseWriter, r *http.Request) {
	manager := backup.NewManager(h.projectDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	backups, err := manager.List(ctx)
	if err != nil {
		h.respondJSON(w, http.StatusInternalServerError, BackupResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to list backups: %v", err),
		})
		return
	}

	// Convert to display format
	displayBackups := make([]BackupDisplay, 0, len(backups))
	for _, b := range backups {
		size, _ := b.GetSize()
		displayBackups = append(displayBackups, BackupDisplay{
			Name:      b.Name,
			Path:      b.Path,
			Size:      size,
			SizeHuman: formatBytes(size),
			Timestamp: b.Metadata.Timestamp,
			Age:       formatAge(b.Metadata.Timestamp),
			Hostname:  b.Metadata.Hostname,
		})
	}

	h.respondJSON(w, http.StatusOK, BackupResponse{
		Success: true,
		Backups: displayBackups,
	})
}

// HandleCreateBackup handles POST /api/backup/create
func (h *IntegrationHandler) HandleCreateBackup(w http.ResponseWriter, r *http.Request) {
	manager := backup.NewManager(h.projectDir)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	b, err := manager.Create(ctx)
	if err != nil {
		h.respondJSON(w, http.StatusInternalServerError, BackupResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to create backup: %v", err),
		})
		return
	}

	size, _ := b.GetSize()

	h.respondJSON(w, http.StatusOK, BackupResponse{
		Success: true,
		Message: "Backup created successfully",
		Backup: &BackupDisplay{
			Name:      b.Name,
			Path:      b.Path,
			Size:      size,
			SizeHuman: formatBytes(size),
			Timestamp: b.Metadata.Timestamp,
			Age:       formatAge(b.Metadata.Timestamp),
			Hostname:  b.Metadata.Hostname,
		},
	})
}

// HandleRestoreBackup handles POST /api/backup/restore
func (h *IntegrationHandler) HandleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	backupName := r.PathValue("name")
	if backupName == "" {
		h.respondJSON(w, http.StatusBadRequest, BackupResponse{
			Success: false,
			Message: "Backup name is required",
		})
		return
	}

	manager := backup.NewManager(h.projectDir)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := manager.Restore(ctx, backupName); err != nil {
		h.respondJSON(w, http.StatusInternalServerError, BackupResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to restore backup: %v", err),
		})
		return
	}

	h.respondJSON(w, http.StatusOK, BackupResponse{
		Success: true,
		Message: "Backup restored successfully. Run 'sdbx down && sdbx up' to apply changes.",
	})
}

// HandleDeleteBackup handles DELETE /api/backup/delete/{name}
func (h *IntegrationHandler) HandleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	backupName := r.PathValue("name")
	if backupName == "" {
		h.respondJSON(w, http.StatusBadRequest, BackupResponse{
			Success: false,
			Message: "Backup name is required",
		})
		return
	}

	manager := backup.NewManager(h.projectDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := manager.Delete(ctx, backupName); err != nil {
		h.respondJSON(w, http.StatusInternalServerError, BackupResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete backup: %v", err),
		})
		return
	}

	h.respondJSON(w, http.StatusOK, BackupResponse{
		Success: true,
		Message: "Backup deleted successfully",
	})
}

// formatBytes formats bytes to human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatAge formats a timestamp as a relative time
func formatAge(t time.Time) string {
	duration := time.Since(t)

	if duration < time.Minute {
		return "just now"
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}
	days := int(duration.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	if days < 30 {
		return fmt.Sprintf("%d days ago", days)
	}

	// For older backups, show full date
	return t.Format("2006-01-02 15:04")
}

// respondJSON sends a JSON response
func (h *IntegrationHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// renderTemplate renders a template with data
func (h *IntegrationHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
