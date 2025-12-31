package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/maiko/sdbx/internal/doctor"
)

func TestDoctorCommand(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-doctor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute doctor command
	if err := runDoctor(doctorCmd, []string{}); err != nil {
		t.Fatalf("runDoctor failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains expected elements
	if !strings.Contains(output, "SDBX Doctor") {
		t.Error("Output should contain 'SDBX Doctor' header")
	}
	if !strings.Contains(output, "checks") {
		t.Error("Output should mention checks")
	}
}

func TestDoctorCommandJSON(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-doctor-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

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

	// Execute doctor command
	if err := runDoctor(doctorCmd, []string{}); err != nil {
		t.Fatalf("runDoctor failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Parse JSON output
	var checks []doctor.Check
	if err := json.Unmarshal([]byte(output), &checks); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify JSON structure
	if len(checks) == 0 {
		t.Error("JSON output should contain at least one check")
	}

	// Verify check structure
	for _, check := range checks {
		if check.Name == "" {
			t.Error("Check should have a name")
		}
		// Status should be set to one of the valid statuses (0-6)
		// Message can be empty for some checks
	}
}

func TestDoctorWithoutProject(t *testing.T) {
	// Create temp directory without project files
	tmpDir, err := os.MkdirTemp("", "sdbx-doctor-noproject-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory (no .sdbx.yaml)
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute doctor command - should not fail even without project
	if err := runDoctor(doctorCmd, []string{}); err != nil {
		t.Fatalf("runDoctor should not fail without project: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should still run checks
	if !strings.Contains(output, "SDBX Doctor") {
		t.Error("Output should contain header even without project")
	}
}
