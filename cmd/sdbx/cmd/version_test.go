package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestVersionInfo(t *testing.T) {
	// Set test version info
	SetVersionInfo("v1.0.0", "abc123", "2025-01-01")

	// Verify values were set
	if Version != "v1.0.0" {
		t.Errorf("Version = %s, want v1.0.0", Version)
	}
	if Commit != "abc123" {
		t.Errorf("Commit = %s, want abc123", Commit)
	}
	if BuildDate != "2025-01-01" {
		t.Errorf("BuildDate = %s, want 2025-01-01", BuildDate)
	}

	// Reset to defaults for other tests
	SetVersionInfo("dev", "none", "unknown")
}

func TestVersionCommandOutput(t *testing.T) {
	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Create pipe to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set test version
	SetVersionInfo("v1.2.3", "deadbeef", "2025-12-31")
	defer SetVersionInfo("dev", "none", "unknown")

	// Execute version command
	if err := versionCmd.RunE(versionCmd, []string{}); err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains expected strings
	if !strings.Contains(output, "v1.2.3") {
		t.Errorf("Output missing version: %s", output)
	}
	if !strings.Contains(output, "deadbeef") {
		t.Errorf("Output missing commit: %s", output)
	}
	if !strings.Contains(output, "2025-12-31") {
		t.Errorf("Output missing build date: %s", output)
	}
}

func TestVersionCommandJSON(t *testing.T) {
	// Save original stdout and json flag
	oldStdout := os.Stdout
	oldJSON := jsonOut
	defer func() {
		os.Stdout = oldStdout
		jsonOut = oldJSON
	}()

	// Enable JSON output
	jsonOut = true

	// Create pipe to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set test version
	SetVersionInfo("v2.0.0", "cafe1234", "2026-01-01")
	defer SetVersionInfo("dev", "none", "unknown")

	// Execute version command
	if err := versionCmd.RunE(versionCmd, []string{}); err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Parse JSON output
	var info VersionInfo
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify JSON fields
	if info.Version != "v2.0.0" {
		t.Errorf("JSON Version = %s, want v2.0.0", info.Version)
	}
	if info.Commit != "cafe1234" {
		t.Errorf("JSON Commit = %s, want cafe1234", info.Commit)
	}
	if info.BuildDate != "2026-01-01" {
		t.Errorf("JSON BuildDate = %s, want 2026-01-01", info.BuildDate)
	}
	if info.GoVersion == "" {
		t.Error("JSON GoVersion should not be empty")
	}
	if info.Platform == "" {
		t.Error("JSON Platform should not be empty")
	}
}
