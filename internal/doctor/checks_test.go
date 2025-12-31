package doctor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDoctor(t *testing.T) {
	doc := NewDoctor("/tmp/test")
	if doc.ProjectDir != "/tmp/test" {
		t.Errorf("ProjectDir = %s, want /tmp/test", doc.ProjectDir)
	}
	if len(doc.Checks) != 0 {
		t.Errorf("Initial checks = %d, want 0", len(doc.Checks))
	}
}

func TestCheckDockerVersion(t *testing.T) {
	doc := NewDoctor(".")
	ctx := context.Background()

	passed, msg := doc.checkDockerVersion(ctx)

	// This test depends on Docker being installed
	// If Docker is installed, it should pass
	// We just verify it doesn't panic
	t.Logf("Docker version check: passed=%v, msg=%s", passed, msg)
}

func TestCheckDiskSpace(t *testing.T) {
	doc := NewDoctor(".")
	ctx := context.Background()

	passed, msg := doc.checkDiskSpace(ctx)

	// Should always return something
	if msg == "" {
		t.Error("Disk space check returned empty message")
	}
	t.Logf("Disk space check: passed=%v, msg=%s", passed, msg)
}

func TestCheckPermissions(t *testing.T) {
	// Create temp dir we know we can write to
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	doc := NewDoctor(tmpDir)
	ctx := context.Background()

	passed, msg := doc.checkPermissions(ctx)
	if !passed {
		t.Errorf("Permissions check failed: %s", msg)
	}
}

func TestCheckProjectFiles(t *testing.T) {
	// Test with no project files
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	doc := NewDoctor(tmpDir)
	ctx := context.Background()

	passed, msg := doc.checkProjectFiles(ctx)
	if passed {
		t.Error("Should fail with no project files")
	}
	if msg == "" {
		t.Error("Should return message about missing files")
	}

	// Create required files
	os.WriteFile(filepath.Join(tmpDir, "compose.yaml"), []byte("version: '3'"), 0o644)
	os.WriteFile(filepath.Join(tmpDir, ".env"), []byte("DOMAIN=test"), 0o644)

	passed, msg = doc.checkProjectFiles(ctx)
	if !passed {
		t.Errorf("Should pass with project files: %s", msg)
	}
}

func TestCheckSecrets(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sdbx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	doc := NewDoctor(tmpDir)
	ctx := context.Background()

	// No secrets dir
	passed, _ := doc.checkSecrets(ctx)
	if passed {
		t.Error("Should fail with no secrets directory")
	}

	// Create secrets dir with files
	secretsDir := filepath.Join(tmpDir, "secrets")
	os.MkdirAll(secretsDir, 0o755)
	os.WriteFile(filepath.Join(secretsDir, "authelia_jwt_secret.txt"), []byte("secret123"), 0o600)
	os.WriteFile(filepath.Join(secretsDir, "authelia_session_secret.txt"), []byte("secret456"), 0o600)

	passed, _ = doc.checkSecrets(ctx)
	if !passed {
		t.Error("Should pass with secrets configured")
	}
}

func TestRunAll(t *testing.T) {
	doc := NewDoctor(".")
	ctx := context.Background()

	checks := doc.RunAll(ctx)

	// Should have run all checks
	if len(checks) == 0 {
		t.Error("RunAll returned no checks")
	}

	// All checks should have a status
	for _, check := range checks {
		if check.Name == "" {
			t.Error("Check has empty name")
		}
		if check.Status == StatusPending || check.Status == StatusRunning {
			t.Errorf("Check %s has invalid status", check.Name)
		}
	}
}
