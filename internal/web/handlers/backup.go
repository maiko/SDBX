package handlers

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"github.com/maiko/sdbx/internal/backup"
)

const (
	backupListTimeout    = 30 * time.Second
	backupCreateTimeout  = 2 * time.Minute
	backupRestoreTimeout = 2 * time.Minute
	backupDeleteTimeout  = 30 * time.Second
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

	ctx, cancel := context.WithTimeout(r.Context(), backupListTimeout)
	defer cancel()

	backups, err := manager.List(ctx)
	if err != nil {
		jsonError(w, "Failed to list backups", "backup.List", err, http.StatusInternalServerError)
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
			SizeHuman: backup.FormatBytes(size),
			Timestamp: b.Metadata.Timestamp,
			Age:       backup.FormatAge(b.Metadata.Timestamp),
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

	ctx, cancel := context.WithTimeout(r.Context(), backupCreateTimeout)
	defer cancel()

	b, err := manager.Create(ctx)
	if err != nil {
		jsonError(w, "Failed to create backup", "backup.Create", err, http.StatusInternalServerError)
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
			SizeHuman: backup.FormatBytes(size),
			Timestamp: b.Metadata.Timestamp,
			Age:       backup.FormatAge(b.Metadata.Timestamp),
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

	if err := backup.ValidateBackupName(backupName); err != nil {
		h.respondJSON(w, http.StatusBadRequest, BackupResponse{
			Success: false,
			Message: "Invalid backup name",
		})
		return
	}

	manager := backup.NewManager(h.projectDir)

	ctx, cancel := context.WithTimeout(r.Context(), backupRestoreTimeout)
	defer cancel()

	if err := manager.Restore(ctx, backupName); err != nil {
		jsonError(w, "Failed to restore backup", "backup.Restore", err, http.StatusInternalServerError)
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

	if err := backup.ValidateBackupName(backupName); err != nil {
		h.respondJSON(w, http.StatusBadRequest, BackupResponse{
			Success: false,
			Message: "Invalid backup name",
		})
		return
	}

	manager := backup.NewManager(h.projectDir)

	ctx, cancel := context.WithTimeout(r.Context(), backupDeleteTimeout)
	defer cancel()

	if err := manager.Delete(ctx, backupName); err != nil {
		jsonError(w, "Failed to delete backup", "backup.Delete", err, http.StatusInternalServerError)
		return
	}

	h.respondJSON(w, http.StatusOK, BackupResponse{
		Success: true,
		Message: "Backup deleted successfully",
	})
}


func (h *BackupHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	respondJSON(w, statusCode, data)
}

func (h *BackupHandler) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	renderTemplate(h.templates, w, name, "backup", data)
}
