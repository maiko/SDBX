package handlers

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestFormatBytes verifies byte formatting
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
		}
	}
}

// TestFormatAge verifies age formatting
func TestFormatAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		contains string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"1 minute", now.Add(-1 * time.Minute), "1 minute ago"},
		{"5 minutes", now.Add(-5 * time.Minute), "5 minutes ago"},
		{"1 hour", now.Add(-1 * time.Hour), "1 hour ago"},
		{"3 hours", now.Add(-3 * time.Hour), "3 hours ago"},
		{"1 day", now.Add(-24 * time.Hour), "1 day ago"},
		{"5 days", now.Add(-5 * 24 * time.Hour), "5 days ago"},
		{"old date", now.Add(-60 * 24 * time.Hour), "-"}, // Shows formatted date with dash separator
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAge(tt.time)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatAge() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

// TestIntegrationHandlerListBackups verifies backup listing
func TestIntegrationHandlerListBackups(t *testing.T) {
	tmpDir := t.TempDir()

	// Create backup directory and a test backup
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	handler := NewIntegrationHandler(tmpDir, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/backup/list", nil)
	w := httptest.NewRecorder()

	handler.HandleListBackups(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if !strings.Contains(body, "success") {
		t.Errorf("response should contain success field")
	}
}

// TestIntegrationHandlerCreateBackup verifies backup creation
func TestIntegrationHandlerCreateBackup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a config file to backup
	configPath := filepath.Join(tmpDir, ".sdbx.yaml")
	if err := os.WriteFile(configPath, []byte("domain: test.local"), 0644); err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	handler := NewIntegrationHandler(tmpDir, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/backup/create", nil)
	w := httptest.NewRecorder()

	handler.HandleCreateBackup(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d, body = %s", w.Code, http.StatusOK, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "success\":true") {
		t.Errorf("response should indicate success, got %s", body)
	}

	if !strings.Contains(body, "Backup created successfully") {
		t.Errorf("response should contain success message")
	}
}

// TestIntegrationHandlerDeleteBackupMissingName verifies delete requires name
func TestIntegrationHandlerDeleteBackupMissingName(t *testing.T) {
	handler := NewIntegrationHandler("", nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/backup/delete/", nil)
	w := httptest.NewRecorder()

	handler.HandleDeleteBackup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Backup name is required") {
		t.Errorf("response should indicate name is required")
	}
}

// TestIntegrationHandlerRestoreBackupMissingName verifies restore requires name
func TestIntegrationHandlerRestoreBackupMissingName(t *testing.T) {
	handler := NewIntegrationHandler("", nil)

	req := httptest.NewRequest(http.MethodPost, "/api/backup/restore/", nil)
	w := httptest.NewRecorder()

	handler.HandleRestoreBackup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Backup name is required") {
		t.Errorf("response should indicate name is required")
	}
}

// TestIntegrationHandlerDeleteNonexistentBackup verifies delete fails for nonexistent
func TestIntegrationHandlerDeleteNonexistentBackup(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewIntegrationHandler(tmpDir, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/backup/delete/{name}", nil)
	req.SetPathValue("name", "nonexistent-backup.tar.gz")
	w := httptest.NewRecorder()

	handler.HandleDeleteBackup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestIntegrationHandlerRestoreNonexistentBackup verifies restore fails for nonexistent
func TestIntegrationHandlerRestoreNonexistentBackup(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewIntegrationHandler(tmpDir, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/backup/restore/{name}", nil)
	req.SetPathValue("name", "nonexistent-backup.tar.gz")
	w := httptest.NewRecorder()

	handler.HandleRestoreBackup(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// TestIntegrationResponseStruct verifies response struct
func TestIntegrationResponseStruct(t *testing.T) {
	resp := IntegrationResponse{
		Success:    true,
		Message:    "test",
		Total:      5,
		Successful: 3,
		Failed:     2,
		Integrations: []IntegrationResultDisplay{
			{Service: "sonarr", Success: true, Message: "ok"},
		},
	}

	if resp.Total != 5 {
		t.Error("Total should be 5")
	}
	if len(resp.Integrations) != 1 {
		t.Error("should have 1 integration")
	}
}

// TestBackupDisplayStruct verifies backup display struct
func TestBackupDisplayStruct(t *testing.T) {
	display := BackupDisplay{
		Name:      "backup.tar.gz",
		Path:      "/path/to/backup.tar.gz",
		Size:      1024,
		SizeHuman: "1.0 KB",
		Timestamp: time.Now(),
		Age:       "just now",
		Hostname:  "testhost",
	}

	if display.Name != "backup.tar.gz" {
		t.Error("Name not set correctly")
	}
	if display.SizeHuman != "1.0 KB" {
		t.Error("SizeHuman not set correctly")
	}
}

// TestBackupResponseStruct verifies backup response struct
func TestBackupResponseStruct(t *testing.T) {
	resp := BackupResponse{
		Success: true,
		Message: "test",
		Backup: &BackupDisplay{
			Name: "backup.tar.gz",
		},
		Backups: []BackupDisplay{
			{Name: "backup1.tar.gz"},
			{Name: "backup2.tar.gz"},
		},
	}

	if !resp.Success {
		t.Error("Success should be true")
	}
	if resp.Backup.Name != "backup.tar.gz" {
		t.Error("Backup name not set correctly")
	}
	if len(resp.Backups) != 2 {
		t.Error("should have 2 backups")
	}
}
