package backup

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewManager verifies manager construction
func TestNewManager(t *testing.T) {
	manager := NewManager("/test/project")

	if manager.projectDir != "/test/project" {
		t.Errorf("expected projectDir '/test/project', got %q", manager.projectDir)
	}

	if manager.backupDir != "/test/project/backups" {
		t.Errorf("expected backupDir '/test/project/backups', got %q", manager.backupDir)
	}
}

// TestCreateBackup verifies backup creation
func TestCreateBackup(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create test files to backup
	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: test.local"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, "secrets"), 0755); err != nil {
		t.Fatalf("failed to create secrets dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "secrets", "jwt.txt"), []byte("secret123"), 0644); err != nil {
		t.Fatalf("failed to create secret file: %v", err)
	}

	manager := NewManager(tmpDir)
	ctx := context.Background()

	backup, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify backup was created
	if backup.Name == "" {
		t.Error("backup name should not be empty")
	}

	if backup.Path == "" {
		t.Error("backup path should not be empty")
	}

	// Check file exists
	if _, err := os.Stat(backup.Path); os.IsNotExist(err) {
		t.Error("backup file should exist")
	}

	// Check metadata
	if backup.Metadata.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", backup.Metadata.Version)
	}

	if backup.Metadata.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

// TestListBackups verifies listing backups
func TestListBackups(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: test.local"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	manager := NewManager(tmpDir)
	ctx := context.Background()

	// Initially no backups
	backups, err := manager.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}

	// Create a backup
	_, err = manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Now should have one backup
	backups, err = manager.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(backups) != 1 {
		t.Errorf("expected 1 backup, got %d", len(backups))
	}
}

// TestListBackupsOrder verifies backups are sorted by timestamp (newest first)
func TestListBackupsOrder(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: test.local"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	manager := NewManager(tmpDir)
	ctx := context.Background()

	// Create multiple backups with delay to ensure different seconds in filename
	_, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Sleep 1.1 seconds to ensure different timestamp in filename
	time.Sleep(1100 * time.Millisecond)

	_, err = manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	backups, err := manager.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(backups) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(backups))
	}

	// First should be newer
	if backups[0].Metadata.Timestamp.Before(backups[1].Metadata.Timestamp) {
		t.Error("backups should be sorted newest first")
	}
}

// TestRestoreBackup verifies backup restoration
func TestRestoreBackup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	configContent := "domain: original.local"
	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	manager := NewManager(tmpDir)
	ctx := context.Background()

	// Create backup
	backup, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Modify the original file
	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: modified.local"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Restore backup
	err = manager.Restore(ctx, backup.Name)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify file was restored
	content, err := os.ReadFile(filepath.Join(tmpDir, ".sdbx.yaml"))
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}

	if string(content) != configContent {
		t.Errorf("expected content %q, got %q", configContent, string(content))
	}
}

// TestRestoreBackupNotFound verifies error when backup doesn't exist
func TestRestoreBackupNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)
	ctx := context.Background()

	err := manager.Restore(ctx, "nonexistent-backup.tar.gz")
	if err == nil {
		t.Error("expected error for nonexistent backup")
	}
}

// TestDeleteBackup verifies backup deletion
func TestDeleteBackup(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: test.local"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	manager := NewManager(tmpDir)
	ctx := context.Background()

	// Create backup
	backup, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify it exists
	backups, _ := manager.List(ctx)
	if len(backups) != 1 {
		t.Fatalf("expected 1 backup, got %d", len(backups))
	}

	// Delete it
	err = manager.Delete(ctx, backup.Name)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	backups, _ = manager.List(ctx)
	if len(backups) != 0 {
		t.Errorf("expected 0 backups after delete, got %d", len(backups))
	}
}

// TestDeleteBackupNotFound verifies error when deleting nonexistent backup
func TestDeleteBackupNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)
	ctx := context.Background()

	err := manager.Delete(ctx, "nonexistent-backup.tar.gz")
	if err == nil {
		t.Error("expected error for nonexistent backup")
	}
}

// TestBackupGetSize verifies getting backup file size
func TestBackupGetSize(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: test.local"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	manager := NewManager(tmpDir)
	ctx := context.Background()

	backup, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	size, err := backup.GetSize()
	if err != nil {
		t.Fatalf("GetSize failed: %v", err)
	}

	if size <= 0 {
		t.Errorf("expected positive size, got %d", size)
	}
}

// TestBackupGetSizeNotFound verifies error for nonexistent backup
func TestBackupGetSizeNotFound(t *testing.T) {
	backup := &Backup{
		Path: "/nonexistent/path/backup.tar.gz",
	}

	_, err := backup.GetSize()
	if err == nil {
		t.Error("expected error for nonexistent backup file")
	}
}

// TestMetadataFields verifies metadata structure
func TestMetadataFields(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: test.local"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	manager := NewManager(tmpDir)
	ctx := context.Background()

	backup, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	metadata := backup.Metadata

	// Version should be set
	if metadata.Version == "" {
		t.Error("metadata version should not be empty")
	}

	// Timestamp should be recent
	if time.Since(metadata.Timestamp) > time.Minute {
		t.Error("metadata timestamp should be recent")
	}

	// ProjectID should be set
	if metadata.ProjectID == "" {
		t.Error("metadata project ID should not be empty")
	}

	// Files list should not be empty
	if len(metadata.Files) == 0 {
		t.Error("metadata files list should not be empty")
	}
}

// TestBackupWithDirectories verifies backup includes directories
func TestBackupWithDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create nested directory structure
	configsDir := filepath.Join(tmpDir, "configs", "sonarr")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create configs dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(configsDir, "config.xml"), []byte("<config/>"), 0644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: test.local"), 0644); err != nil {
		t.Fatalf("failed to create yaml file: %v", err)
	}

	manager := NewManager(tmpDir)
	ctx := context.Background()

	backup, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Delete configs to verify restore
	if err := os.RemoveAll(filepath.Join(tmpDir, "configs")); err != nil {
		t.Fatalf("failed to remove configs: %v", err)
	}

	// Restore
	err = manager.Restore(ctx, backup.Name)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Verify nested file was restored
	content, err := os.ReadFile(filepath.Join(configsDir, "config.xml"))
	if err != nil {
		t.Fatalf("failed to read restored config: %v", err)
	}

	if string(content) != "<config/>" {
		t.Errorf("expected '<config/>', got %q", string(content))
	}
}

// TestListEmptyBackupDir verifies list works with no backup directory
func TestListEmptyBackupDir(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)
	ctx := context.Background()

	// Backup directory doesn't exist yet
	backups, err := manager.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(backups) != 0 {
		t.Errorf("expected 0 backups, got %d", len(backups))
	}
}

// TestBackupSkipsMissingFiles verifies backup doesn't fail on missing files
func TestBackupSkipsMissingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Only create .sdbx.yaml, not other files
	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: test.local"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	manager := NewManager(tmpDir)
	ctx := context.Background()

	// Should not fail even though compose.yaml, secrets/, configs/ don't exist
	backup, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if backup == nil {
		t.Error("backup should not be nil")
	}
}
