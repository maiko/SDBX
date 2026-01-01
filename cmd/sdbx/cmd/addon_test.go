package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maiko/sdbx/internal/config"
)

func TestAddonList(t *testing.T) {
	// SKIP: This test requires Git source setup (addons are no longer embedded)
	// TODO: Refactor to use test fixtures or mock registry
	t.Skip("Addon tests require Git source configuration - skipping for now")

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-addon-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create test config with some addons enabled
	cfg := config.DefaultConfig()
	cfg.EnableAddon("overseerr")
	cfg.EnableAddon("tautulli")
	if err := cfg.Save(".sdbx.yaml"); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute list command
	if err := runAddonList(addonListCmd, []string{}); err != nil {
		t.Fatalf("runAddonList failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains addon names
	if !strings.Contains(output, "overseerr") {
		t.Error("Output should contain 'overseerr'")
	}
	if !strings.Contains(output, "tautulli") {
		t.Error("Output should contain 'tautulli'")
	}
	if !strings.Contains(output, "enabled") {
		t.Error("Output should contain 'enabled'")
	}
}

func TestAddonListJSON(t *testing.T) {
	// SKIP: This test requires Git source setup (addons are no longer embedded)
	// TODO: Refactor to use test fixtures or mock registry
	t.Skip("Addon tests require Git source configuration - skipping for now")

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-addon-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create test config with addon enabled
	cfg := config.DefaultConfig()
	cfg.EnableAddon("wizarr")
	if err := cfg.Save(".sdbx.yaml"); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
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
	if err := runAddonList(addonListCmd, []string{}); err != nil {
		t.Fatalf("runAddonList failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Parse JSON output
	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, output)
	}

	// Verify JSON structure - without --all flag, only enabled addons are shown
	// In this test, only wizarr is enabled
	if len(result) != 1 {
		t.Errorf("JSON result length = %d, want 1 (only enabled addons)", len(result))
	}

	// Find wizarr in results
	foundWizarr := false
	for _, addon := range result {
		if addon["name"] == "wizarr" {
			foundWizarr = true
			if addon["enabled"] != true {
				t.Error("wizarr should be enabled in JSON output")
			}
		}
	}
	if !foundWizarr {
		t.Error("wizarr not found in JSON output")
	}
}

func TestAddonEnable(t *testing.T) {
	// SKIP: This test requires Git source setup (addons are no longer embedded)
	// TODO: Refactor to use test fixtures or mock registry
	t.Skip("Addon tests require Git source configuration - skipping for now")

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-addon-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create test config
	cfg := config.DefaultConfig()
	if err := cfg.Save(".sdbx.yaml"); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute enable command
	if err := runAddonEnable(addonEnableCmd, []string{"lidarr"}); err != nil {
		t.Fatalf("runAddonEnable failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output
	if !strings.Contains(output, "Enabled") || !strings.Contains(output, "lidarr") {
		t.Errorf("Output should confirm addon enabled: %s", output)
	}

	// Verify config was saved
	cfgPath := filepath.Join(tmpDir, ".sdbx.yaml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Error("Config file should exist after enabling addon")
	}

	// Load config and verify addon is enabled
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if !loadedCfg.IsAddonEnabled("lidarr") {
		t.Error("lidarr should be enabled in saved config")
	}
}

func TestAddonEnableInvalid(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-addon-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create test config
	cfg := config.DefaultConfig()
	if err := cfg.Save(".sdbx.yaml"); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Try to enable invalid addon
	err = runAddonEnable(addonEnableCmd, []string{"nonexistent"})
	if err == nil {
		t.Error("runAddonEnable should fail for invalid addon")
	}
	if !strings.Contains(err.Error(), "addon not found") {
		t.Errorf("Error should mention addon not found: %v", err)
	}
}

func TestAddonEnableAlreadyEnabled(t *testing.T) {
	// SKIP: This test requires Git source setup (addons are no longer embedded)
	// TODO: Refactor to use test fixtures or mock registry
	t.Skip("Addon tests require Git source configuration - skipping for now")

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-addon-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create test config with addon already enabled
	cfg := config.DefaultConfig()
	cfg.EnableAddon("bazarr")
	if err := cfg.Save(".sdbx.yaml"); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Try to enable already enabled addon
	if err := runAddonEnable(addonEnableCmd, []string{"bazarr"}); err != nil {
		t.Fatalf("runAddonEnable should not fail for already enabled addon: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output mentions already enabled
	if !strings.Contains(output, "already enabled") {
		t.Errorf("Output should mention already enabled: %s", output)
	}
}

func TestAddonDisable(t *testing.T) {
	// SKIP: This test requires Git source setup (addons are no longer embedded)
	// TODO: Refactor to use test fixtures or mock registry
	t.Skip("Addon tests require Git source configuration - skipping for now")

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-addon-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create test config with addon enabled
	cfg := config.DefaultConfig()
	cfg.EnableAddon("readarr")
	if err := cfg.Save(".sdbx.yaml"); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute disable command
	if err := runAddonDisable(addonDisableCmd, []string{"readarr"}); err != nil {
		t.Fatalf("runAddonDisable failed: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output
	if !strings.Contains(output, "Disabled") || !strings.Contains(output, "readarr") {
		t.Errorf("Output should confirm addon disabled: %s", output)
	}

	// Load config and verify addon is disabled
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if loadedCfg.IsAddonEnabled("readarr") {
		t.Error("readarr should be disabled in saved config")
	}
}

func TestAddonDisableNotEnabled(t *testing.T) {
	// SKIP: This test requires Git source setup (addons are no longer embedded)
	// TODO: Refactor to use test fixtures or mock registry
	t.Skip("Addon tests require Git source configuration - skipping for now")

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-addon-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Change to temp directory
	oldCwd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldCwd)

	// Create test config without addon
	cfg := config.DefaultConfig()
	if err := cfg.Save(".sdbx.yaml"); err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Try to disable non-enabled addon
	if err := runAddonDisable(addonDisableCmd, []string{"flaresolverr"}); err != nil {
		t.Fatalf("runAddonDisable should not fail for non-enabled addon: %v", err)
	}

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output mentions not enabled
	if !strings.Contains(output, "not enabled") {
		t.Errorf("Output should mention not enabled: %s", output)
	}
}
