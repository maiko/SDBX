package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/maiko/sdbx/internal/backup"
)

// BackupHandler handles backup and restore operations
type BackupHandler struct {
	projectDir string
	templates  *template.Template
}

// NewBackupHandler creates a new backup handler
func NewBackupHandler(projectDir string, tmpl *template.Template) *BackupHandler {
	return &BackupHandler{
		projectDir: projectDir,
		templates:  tmpl,
	}
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

// HandleBackupPage handles the backup management page
func (h *BackupHandler) HandleBackupPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"ProjectDir": h.projectDir,
	}

	h.renderTemplate(w, "pages/backup.html", data)
}

// HandleListBackups handles GET /api/backup/list
func (h *BackupHandler) HandleListBackups(w http.ResponseWriter, r *http.Request) {
	manager := backup.NewManager(h.projectDir)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
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
func (h *BackupHandler) HandleCreateBackup(w http.ResponseWriter, r *http.Request) {
	manager := backup.NewManager(h.projectDir)

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
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
func (h *BackupHandler) HandleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	backupName := r.PathValue("name")
	if backupName == "" {
		h.respondJSON(w, http.StatusBadRequest, BackupResponse{
			Success: false,
			Message: "Backup name is required",
		})
		return
	}

	manager := backup.NewManager(h.projectDir)

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
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
func (h *BackupHandler) HandleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	backupName := r.PathValue("name")
	if backupName == "" {
		h.respondJSON(w, http.StatusBadRequest, BackupResponse{
			Success: false,
			Message: "Backup name is required",
		})
		return
	}

	manager := backup.NewManager(h.projectDir)

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
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
func (h *BackupHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// renderTemplate renders a template with data
func (h *BackupHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		httpError(w, "backup template render", err, http.StatusInternalServerError)
	}
}
