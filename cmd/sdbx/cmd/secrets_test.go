package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSecretsGenerate(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-secrets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create secrets directory
	secretsDir := filepath.Join(tmpDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		t.Fatalf("Failed to create secrets dir: %v", err)
	}

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute generate command
	if err := runSecretsGenerate(secretsGenerateCmd, []string{}); err != nil {
		t.Fatalf("runSecretsGenerate failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output
	if !strings.Contains(output, "Secrets generated") {
		t.Errorf("Output should confirm secrets generated: %s", output)
	}

	// Verify secrets were created
	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		t.Fatalf("Failed to read secrets dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("Secrets directory should contain files")
	}
}

func TestSecretsList(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-secrets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create secrets directory with test secrets
	secretsDir := filepath.Join(tmpDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		t.Fatalf("Failed to create secrets dir: %v", err)
	}

	// Create a configured secret
	if err := os.WriteFile(filepath.Join(secretsDir, "test_secret.txt"), []byte("test_value"), 0o600); err != nil {
		t.Fatalf("Failed to write test secret: %v", err)
	}

	// Create an empty secret
	if err := os.WriteFile(filepath.Join(secretsDir, "empty_secret.txt"), []byte(""), 0o600); err != nil {
		t.Fatalf("Failed to write empty secret: %v", err)
	}

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute list command
	if err := runSecretsList(secretsListCmd, []string{}); err != nil {
		t.Fatalf("runSecretsList failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains header
	if !strings.Contains(output, "SDBX Secrets") {
		t.Error("Output should contain header")
	}
}

func TestSecretsListJSON(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-secrets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create secrets directory with test secrets
	secretsDir := filepath.Join(tmpDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		t.Fatalf("Failed to create secrets dir: %v", err)
	}

	// Create a configured secret
	if err := os.WriteFile(filepath.Join(secretsDir, "configured.txt"), []byte("value"), 0o600); err != nil {
		t.Fatalf("Failed to write test secret: %v", err)
	}

	// Save original stdout and json flag
	oldStdout := os.Stdout
	oldJSON := jsonOut
	defer func() {
		os.Stdout = oldStdout
		jsonOut = oldJSON
	}()

	// Enable JSON output
	jsonOut = true

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute list command
	if err := runSecretsList(secretsListCmd, []string{}); err != nil {
		t.Fatalf("runSecretsList failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Parse JSON output
	var status map[string]bool
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify JSON structure
	if len(status) == 0 {
		t.Error("JSON output should contain secrets")
	}
}

func TestSecretsRotate(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-secrets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create secrets directory
	secretsDir := filepath.Join(tmpDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		t.Fatalf("Failed to create secrets dir: %v", err)
	}

	// Create an existing secret
	secretFile := "authelia_jwt_secret.txt"
	secretPath := filepath.Join(secretsDir, secretFile)
	originalValue := "original_secret_value_123456"
	if err := os.WriteFile(secretPath, []byte(originalValue), 0o600); err != nil {
		t.Fatalf("Failed to write test secret: %v", err)
	}

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute rotate command
	if err := runSecretsRotate(secretsRotateCmd, []string{secretFile}); err != nil {
		t.Fatalf("runSecretsRotate failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output
	if !strings.Contains(output, "Rotated") {
		t.Errorf("Output should confirm rotation: %s", output)
	}

	// Verify secret was changed
	newValue, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("Failed to read rotated secret: %v", err)
	}
	if string(newValue) == originalValue {
		t.Error("Secret value should have changed after rotation")
	}

	// Verify backup was created
	backups, err := filepath.Glob(filepath.Join(secretsDir, secretFile+".backup.*"))
	if err != nil {
		t.Fatalf("Failed to find backups: %v", err)
	}
	if len(backups) == 0 {
		t.Error("Backup should have been created")
	}
}

func TestSecretsRotateManual(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-secrets-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create secrets directory
	secretsDir := filepath.Join(tmpDir, "secrets")
	if err := os.MkdirAll(secretsDir, 0o755); err != nil {
		t.Fatalf("Failed to create secrets dir: %v", err)
	}

	// Try to rotate a manual secret (should fail)
	err = runSecretsRotate(secretsRotateCmd, []string{"vpn_password.txt"})
	if err == nil {
		t.Error("Rotating manual secret should fail")
	}
	if !strings.Contains(err.Error(), "manual") {
		t.Errorf("Error should mention manual configuration: %v", err)
	}
}
