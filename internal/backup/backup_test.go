package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"strings"
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

	// Sleep to ensure safety backup gets a different timestamp-based filename
	time.Sleep(1100 * time.Millisecond)

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

	// Sleep to ensure safety backup gets a different timestamp-based filename
	time.Sleep(1100 * time.Millisecond)

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

// TestValidateBackupName verifies backup name validation
func TestValidateBackupName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"sdbx-backup-2026-01-01-120000.tar.gz", false},
		{"my-backup.tar.gz", false},
		{"", true},
		{"..", true},
		{"../etc/passwd", true},
		{"foo/../bar", true},
		{"/etc/passwd", true},
		{"subdir/backup.tar.gz", true},
		{"back\\slash.tar.gz", true},
	}

	for _, tt := range tests {
		err := ValidateBackupName(tt.name)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateBackupName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

// TestRestoreRejectsPathTraversalInTar verifies that tar entries with .. are rejected
func TestRestoreRejectsPathTraversalInTar(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)
	ctx := context.Background()

	// Create a malicious tar.gz with a path traversal entry
	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	archivePath := filepath.Join(backupDir, "malicious.tar.gz")
	createMaliciousTar(t, archivePath, "../../../etc/evil", "pwned")

	err := manager.Restore(ctx, "malicious.tar.gz")
	if err == nil {
		t.Fatal("Restore should reject tar entries with path traversal")
	}

	if !strings.Contains(err.Error(), "unsafe path") {
		t.Errorf("error should mention unsafe path, got: %v", err)
	}

	// Verify the evil file was NOT created
	if _, err := os.Stat(filepath.Join(tmpDir, "..", "..", "..", "etc", "evil")); err == nil {
		t.Fatal("path traversal file should not have been created")
	}
}

// TestRestoreRejectsAbsolutePathInTar verifies that absolute paths in tar entries are rejected
func TestRestoreRejectsAbsolutePathInTar(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)
	ctx := context.Background()

	backupDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	archivePath := filepath.Join(backupDir, "abs-path.tar.gz")
	createMaliciousTar(t, archivePath, "/tmp/evil", "pwned")

	err := manager.Restore(ctx, "abs-path.tar.gz")
	if err == nil {
		t.Fatal("Restore should reject tar entries with absolute paths")
	}

	if !strings.Contains(err.Error(), "unsafe path") {
		t.Errorf("error should mention unsafe path, got: %v", err)
	}
}

// TestRestoreRejectsTraversalBackupName verifies that backup names with traversal are rejected
func TestRestoreRejectsTraversalBackupName(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)
	ctx := context.Background()

	err := manager.Restore(ctx, "../../../etc/passwd")
	if err == nil {
		t.Fatal("Restore should reject backup names with path traversal")
	}

	if !strings.Contains(err.Error(), "invalid backup name") {
		t.Errorf("error should mention invalid backup name, got: %v", err)
	}
}

// TestDeleteRejectsTraversalBackupName verifies that delete rejects traversal names
func TestDeleteRejectsTraversalBackupName(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)
	ctx := context.Background()

	err := manager.Delete(ctx, "../../../etc/important")
	if err == nil {
		t.Fatal("Delete should reject backup names with path traversal")
	}

	if !strings.Contains(err.Error(), "invalid backup name") {
		t.Errorf("error should mention invalid backup name, got: %v", err)
	}
}

// TestRestoreSafeEntriesStillWork verifies that legitimate tar entries are still extracted
func TestRestoreSafeEntriesStillWork(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: safe.local"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	manager := NewManager(tmpDir)
	ctx := context.Background()

	// Create a legitimate backup
	backup, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Modify the file
	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: changed.local"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Sleep to ensure safety backup gets a different timestamp-based filename
	time.Sleep(1100 * time.Millisecond)

	// Restore should work with clean tar entries
	if err := manager.Restore(ctx, backup.Name); err != nil {
		t.Fatalf("Restore of clean backup failed: %v", err)
	}

	// Verify file was restored
	content, err := os.ReadFile(filepath.Join(tmpDir, ".sdbx.yaml"))
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}

	if string(content) != "domain: safe.local" {
		t.Errorf("expected 'domain: safe.local', got %q", string(content))
	}
}

// createMaliciousTar creates a tar.gz archive with a specific entry name (for testing)
func createMaliciousTar(t *testing.T, archivePath, entryName, content string) {
	t.Helper()

	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer f.Close()

	gzWriter := gzip.NewWriter(f)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Write metadata.json first (required by Restore)
	metadataContent := `{"version":"1.0.0","timestamp":"2026-01-01T00:00:00Z"}`
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "metadata.json",
		Mode: 0644,
		Size: int64(len(metadataContent)),
	}); err != nil {
		t.Fatalf("failed to write metadata header: %v", err)
	}
	if _, err := tarWriter.Write([]byte(metadataContent)); err != nil {
		t.Fatalf("failed to write metadata: %v", err)
	}

	// Write malicious entry
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: entryName,
		Mode: 0644,
		Size: int64(len(content)),
	}); err != nil {
		t.Fatalf("failed to write malicious header: %v", err)
	}
	if _, err := tarWriter.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write malicious content: %v", err)
	}
}

// TestBackupSkipsLargeFiles verifies files >100MB are skipped during archive creation
func TestBackupSkipsLargeFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .sdbx.yaml
	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: test.local"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a configs directory with a small file and a large file (>100MB)
	configsDir := filepath.Join(tmpDir, "configs", "sonarr")
	if err := os.MkdirAll(configsDir, 0755); err != nil {
		t.Fatalf("failed to create configs dir: %v", err)
	}

	// Small file - should be included
	smallContent := []byte("small config file")
	if err := os.WriteFile(filepath.Join(configsDir, "config.xml"), smallContent, 0644); err != nil {
		t.Fatalf("failed to create small file: %v", err)
	}

	// Create a file just over 100MB using a sparse file
	largePath := filepath.Join(configsDir, "sonarr.db")
	largeFile, err := os.Create(largePath)
	if err != nil {
		t.Fatalf("failed to create large file: %v", err)
	}
	// Seek to 101MB and write a byte to create a sparse file (fast, uses minimal disk)
	const largeSize = 101 << 20 // 101 MiB
	if _, err := largeFile.Seek(largeSize-1, 0); err != nil {
		largeFile.Close()
		t.Fatalf("failed to seek: %v", err)
	}
	if _, err := largeFile.Write([]byte{0}); err != nil {
		largeFile.Close()
		t.Fatalf("failed to write: %v", err)
	}
	largeFile.Close()

	manager := NewManager(tmpDir)
	ctx := context.Background()

	backup, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Read the archive and check which files are included
	f, err := os.Open(backup.Path)
	if err != nil {
		t.Fatalf("failed to open backup: %v", err)
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var foundSmall, foundLarge bool
	for {
		header, err := tarReader.Next()
		if err != nil {
			break
		}
		if strings.Contains(header.Name, "config.xml") {
			foundSmall = true
		}
		if strings.Contains(header.Name, "sonarr.db") {
			foundLarge = true
		}
	}

	if !foundSmall {
		t.Error("small file (config.xml) should be included in backup")
	}
	if foundLarge {
		t.Error("large file (sonarr.db >100MB) should be skipped in backup")
	}
}

// TestRestoreCreatesSafetyBackup verifies that a safety backup is created before restore
func TestRestoreCreatesSafetyBackup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: original.local"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	manager := NewManager(tmpDir)
	ctx := context.Background()

	// Create initial backup
	backup, err := manager.Create(ctx)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Modify the file so the safety backup captures different content
	if err := os.WriteFile(filepath.Join(tmpDir, ".sdbx.yaml"), []byte("domain: modified.local"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	// Count backups before restore
	backupsBefore, err := manager.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	countBefore := len(backupsBefore)

	// Sleep to ensure different timestamp in filename
	time.Sleep(1100 * time.Millisecond)

	// Restore should create a safety backup
	err = manager.Restore(ctx, backup.Name)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}

	// Count backups after restore - should have one more (the safety backup)
	backupsAfter, err := manager.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(backupsAfter) != countBefore+1 {
		t.Errorf("expected %d backups after restore (safety backup created), got %d", countBefore+1, len(backupsAfter))
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
