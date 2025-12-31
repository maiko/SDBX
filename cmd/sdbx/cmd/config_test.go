package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestConfigGetAll(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Set up test config
	viper.Reset()
	viper.Set("domain", "test.example.com")
	viper.Set("timezone", "UTC")

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute get command (no args = get all)
	if err := runConfigGet(configGetCmd, []string{}); err != nil {
		t.Fatalf("runConfigGet failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains title
	if !strings.Contains(output, "SDBX Configuration") {
		t.Error("Output should contain configuration title")
	}

	// Verify test values appear
	if !strings.Contains(output, "test.example.com") {
		t.Error("Output should contain domain value")
	}

	// Reset viper
	viper.Reset()
}

func TestConfigGetSingle(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Set up test config
	viper.Reset()
	viper.Set("domain", "single.example.com")

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute get command for single key
	if err := runConfigGet(configGetCmd, []string{"domain"}); err != nil {
		t.Fatalf("runConfigGet failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains key=value format
	if !strings.Contains(output, "domain = single.example.com") {
		t.Errorf("Output should show domain value: %s", output)
	}

	// Reset viper
	viper.Reset()
}

func TestConfigGetUnknownKey(t *testing.T) {
	// Set up test config
	viper.Reset()
	defer viper.Reset()

	// Try to get unknown key
	err := runConfigGet(configGetCmd, []string{"nonexistent_key"})
	if err == nil {
		t.Error("runConfigGet should fail for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown configuration key") {
		t.Errorf("Error should mention unknown key: %v", err)
	}
}

func TestConfigGetJSON(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Set up test config
	viper.Reset()
	viper.Set("domain", "json.example.com")
	viper.Set("timezone", "America/New_York")

	// Save original stdout and json flag
	oldStdout := os.Stdout
	oldJSON := jsonOut
	defer func() {
		os.Stdout = oldStdout
		jsonOut = oldJSON
		viper.Reset()
	}()

	// Enable JSON output
	jsonOut = true

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute get command
	if err := runConfigGet(configGetCmd, []string{}); err != nil {
		t.Fatalf("runConfigGet failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Parse JSON output
	var config map[string]interface{}
	if err := json.Unmarshal([]byte(output), &config); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify JSON contains expected values
	if config["domain"] != "json.example.com" {
		t.Errorf("JSON domain = %v, want json.example.com", config["domain"])
	}
}

func TestConfigSet(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Reset viper
	viper.Reset()
	defer viper.Reset()

	// Set config file location
	viper.SetConfigFile(".sdbx.yaml")

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute set command
	if err := runConfigSet(configSetCmd, []string{"domain", "newdomain.example.com"}); err != nil {
		t.Fatalf("runConfigSet failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output
	if !strings.Contains(output, "Set domain = newdomain.example.com") {
		t.Errorf("Output should confirm set: %s", output)
	}

	// Verify value was actually set
	value := viper.GetString("domain")
	if value != "newdomain.example.com" {
		t.Errorf("Domain value = %s, want newdomain.example.com", value)
	}
}

func TestConfigSetInvalidKey(t *testing.T) {
	// Reset viper
	viper.Reset()
	defer viper.Reset()

	// Try to set invalid key
	err := runConfigSet(configSetCmd, []string{"invalid_key", "value"})
	if err == nil {
		t.Error("runConfigSet should fail for invalid key")
	}
	if !strings.Contains(err.Error(), "invalid configuration key") {
		t.Errorf("Error should mention invalid key: %v", err)
	}
}
