package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"short", 16},
		{"medium", 32},
		{"long", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateRandomString(tt.length)
			if err != nil {
				t.Fatalf("GenerateRandomString(%d) error = %v", tt.length, err)
			}
			if len(result) != tt.length {
				t.Errorf("GenerateRandomString(%d) = %d chars, want %d", tt.length, len(result), tt.length)
			}
		})
	}
}

func TestGenerateRandomStringUniqueness(t *testing.T) {
	// Generate multiple strings and ensure they're different
	results := make(map[string]bool)
	for i := 0; i < 100; i++ {
		s, err := GenerateRandomString(32)
		if err != nil {
			t.Fatalf("GenerateRandomString failed: %v", err)
		}
		if results[s] {
			t.Error("GenerateRandomString produced duplicate value")
		}
		results[s] = true
	}
}

func TestGenerateSecrets(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate secrets
	if err := GenerateSecrets(tmpDir); err != nil {
		t.Fatalf("GenerateSecrets failed: %v", err)
	}

	// Verify files were created
	for filename, expectedLen := range SecretFiles {
		path := filepath.Join(tmpDir, filename)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("Secret file %s not created: %v", filename, err)
			continue
		}

		// User-provided secrets should be empty
		if expectedLen == 0 {
			if info.Size() != 0 {
				t.Errorf("Secret file %s should be empty", filename)
			}
			continue
		}

		// Auto-generated secrets should have content
		if info.Size() == 0 {
			t.Errorf("Secret file %s should have content", filename)
		}
	}
}

func TestRotateSecret(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate initial secrets
	if err := GenerateSecrets(tmpDir); err != nil {
		t.Fatalf("GenerateSecrets failed: %v", err)
	}

	// Read original value
	original, _ := ReadSecret(tmpDir, "authelia_jwt_secret.txt")

	// Rotate
	newSecret, err := RotateSecret(tmpDir, "authelia_jwt_secret.txt")
	if err != nil {
		t.Fatalf("RotateSecret failed: %v", err)
	}

	// Verify it changed
	if newSecret == original {
		t.Error("RotateSecret did not change the secret")
	}

	// Verify file was updated
	current, _ := ReadSecret(tmpDir, "authelia_jwt_secret.txt")
	if current != newSecret {
		t.Error("RotateSecret did not update file")
	}
}

func TestRotateUserProvidedSecret(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Try to rotate user-provided secret
	_, err = RotateSecret(tmpDir, "vpn_password.txt")
	if err == nil {
		t.Error("RotateSecret should fail for user-provided secrets")
	}
}

func TestListSecrets(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate secrets
	if err := GenerateSecrets(tmpDir); err != nil {
		t.Fatalf("GenerateSecrets failed: %v", err)
	}

	// List secrets
	status, err := ListSecrets(tmpDir)
	if err != nil {
		t.Fatalf("ListSecrets failed: %v", err)
	}

	// Verify all secrets are listed
	for filename := range SecretFiles {
		if _, ok := status[filename]; !ok {
			t.Errorf("Secret %s not in list", filename)
		}
	}
}

func TestReadSecretNotConfigured(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create empty file
	emptyFile := filepath.Join(tmpDir, "empty_secret.txt")
	if err := os.WriteFile(emptyFile, []byte(""), 0o600); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	// Try to read empty secret
	_, err = ReadSecret(tmpDir, "empty_secret.txt")
	if err == nil {
		t.Error("ReadSecret should fail for empty secret")
	}

	if !IsSecretNotConfigured(err) {
		t.Errorf("Expected SecretNotConfiguredError, got: %v", err)
	}
}

func TestRotateSecretBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate initial secrets
	if err := GenerateSecrets(tmpDir); err != nil {
		t.Fatalf("GenerateSecrets failed: %v", err)
	}

	// Read original value
	original, _ := ReadSecret(tmpDir, "authelia_jwt_secret.txt")

	// Rotate secret
	newSecret, err := RotateSecret(tmpDir, "authelia_jwt_secret.txt")
	if err != nil {
		t.Fatalf("RotateSecret failed: %v", err)
	}

	// Verify backup was created
	backups, err := filepath.Glob(filepath.Join(tmpDir, "authelia_jwt_secret.txt.backup.*"))
	if err != nil {
		t.Fatalf("Failed to find backups: %v", err)
	}
	if len(backups) == 0 {
		t.Error("No backup file was created")
	}

	// Verify backup contains original value
	if len(backups) > 0 {
		backupContent, err := os.ReadFile(backups[0])
		if err != nil {
			t.Fatalf("Failed to read backup: %v", err)
		}
		if string(backupContent) != original {
			t.Error("Backup does not contain original value")
		}
	}

	// Verify new secret is different
	if newSecret == original {
		t.Error("RotateSecret did not change the secret")
	}

	// Verify new secret was written
	current, _ := ReadSecret(tmpDir, "authelia_jwt_secret.txt")
	if strings.TrimSpace(current) != strings.TrimSpace(newSecret) {
		t.Error("RotateSecret did not update file")
	}
}

func TestManualSecretError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Try to rotate user-provided secret
	_, err = RotateSecret(tmpDir, "vpn_password.txt")
	if err == nil {
		t.Error("RotateSecret should fail for user-provided secrets")
	}

	if !IsManualSecret(err) {
		t.Errorf("Expected ManualSecretError, got: %v", err)
	}
}

func TestCleanupBackups(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some backup files with different ages
	oldBackup := filepath.Join(tmpDir, "secret.txt.backup.1000000000")
	recentBackup := filepath.Join(tmpDir, "secret.txt.backup.9999999999")

	if err := os.WriteFile(oldBackup, []byte("old"), 0o600); err != nil {
		t.Fatalf("Failed to create old backup: %v", err)
	}
	if err := os.WriteFile(recentBackup, []byte("recent"), 0o600); err != nil {
		t.Fatalf("Failed to create recent backup: %v", err)
	}

	// Set old backup's modification time to past
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(oldBackup, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to set old backup time: %v", err)
	}

	// Cleanup backups older than 24 hours
	if err := CleanupBackups(tmpDir, 24*time.Hour); err != nil {
		t.Fatalf("CleanupBackups failed: %v", err)
	}

	// Verify old backup was removed
	if _, err := os.Stat(oldBackup); !os.IsNotExist(err) {
		t.Error("Old backup should have been removed")
	}

	// Verify recent backup still exists
	if _, err := os.Stat(recentBackup); err != nil {
		t.Error("Recent backup should still exist")
	}
}

func TestSecretErrors(t *testing.T) {
	// Test SecretNotConfiguredError
	err := &SecretNotConfiguredError{Filename: "test.txt"}
	if !IsSecretNotConfigured(err) {
		t.Error("IsSecretNotConfigured should return true")
	}

	// Test ManualSecretError
	err2 := &ManualSecretError{Filename: "vpn.txt"}
	if !IsManualSecret(err2) {
		t.Error("IsManualSecret should return true")
	}
}

func TestRotateAllSecrets(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate initial secrets
	if err := GenerateSecrets(tmpDir); err != nil {
		t.Fatalf("GenerateSecrets failed: %v", err)
	}

	// Rotate all secrets
	results, err := RotateAllSecrets(tmpDir)
	if err != nil {
		t.Fatalf("RotateAllSecrets failed: %v", err)
	}

	// Verify all auto-generated secrets were rotated
	for filename, length := range SecretFiles {
		if length > 0 { // Only auto-generated
			if _, ok := results[filename]; !ok {
				t.Errorf("Secret %s was not rotated", filename)
			}
		}
	}

	// Verify user-provided secrets were skipped
	if _, ok := results["vpn_password.txt"]; ok {
		t.Error("User-provided secret should not be in results")
	}
}
