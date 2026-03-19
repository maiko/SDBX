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
	"github.com/maiko/sdbx/internal/registry"
)

// testAddonYAML returns a minimal service YAML for a test addon
func testAddonYAML(name, category, description string) string {
	return `apiVersion: sdbx.io/v1
kind: Service
metadata:
  name: ` + name + `
  version: 1.0.0
  category: ` + category + `
  description: "` + description + `"
spec:
  image:
    repository: linuxserver/` + name + `
    tag: latest
  container:
    name_template: "sdbx-{{ .Name }}"
    restart: unless-stopped
routing:
  enabled: true
  port: 8080
  subdomain: ` + name + `
  path: /` + name + `
  auth:
    required: true
conditions:
  requireAddon: true
`
}

// setupTestRegistry creates a temp directory with test addon definitions
// and overrides the registryProvider to use it. Returns a cleanup function.
func setupTestRegistry(t *testing.T, addons map[string]string) func() {
	t.Helper()

	tmpDir := t.TempDir()
	addonsDir := filepath.Join(tmpDir, "addons")

	for name, yaml := range addons {
		dir := filepath.Join(addonsDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create addon dir %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "service.yaml"), []byte(yaml), 0644); err != nil {
			t.Fatalf("failed to write service.yaml for %s: %v", name, err)
		}
	}

	oldProvider := registryProvider
	registryProvider = func() (*registry.Registry, error) {
		cfg := &registry.SourceConfig{
			Sources: []registry.Source{
				{
					Name:     "test-addons",
					Type:     "local",
					Path:     tmpDir,
					Enabled:  true,
					Priority: 100,
				},
			},
			Cache: registry.CacheConfig{
				Directory: t.TempDir(),
			},
		}
		return registry.New(cfg)
	}

	return func() {
		registryProvider = oldProvider
	}
}

// defaultTestAddons returns a standard set of test addon definitions
func defaultTestAddons() map[string]string {
	return map[string]string{
		"overseerr": testAddonYAML("overseerr", "media", "Media request management"),
		"tautulli":  testAddonYAML("tautulli", "media", "Plex monitoring and statistics"),
		"lidarr":    testAddonYAML("lidarr", "media", "Music automation and management"),
		"bazarr":    testAddonYAML("bazarr", "media", "Subtitle automation"),
		"readarr":   testAddonYAML("readarr", "media", "Book automation"),
		"wizarr":    testAddonYAML("wizarr", "utility", "Plex invitation management"),
	}
}

func TestAddonList(t *testing.T) {
	cleanup := setupTestRegistry(t, defaultTestAddons())
	defer cleanup()

	// Create temp directory for test
	tmpDir := t.TempDir()

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
		w.Close()
		os.Stdout = oldStdout
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
	cleanup := setupTestRegistry(t, defaultTestAddons())
	defer cleanup()

	// Create temp directory for test
	tmpDir := t.TempDir()

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
		w.Close()
		os.Stdout = oldStdout
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
	cleanup := setupTestRegistry(t, defaultTestAddons())
	defer cleanup()

	// Create temp directory for test
	tmpDir := t.TempDir()

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
		w.Close()
		os.Stdout = oldStdout
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
	tmpDir := t.TempDir()

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
	err := runAddonEnable(addonEnableCmd, []string{"nonexistent"})
	if err == nil {
		t.Error("runAddonEnable should fail for invalid addon")
	}
	if !strings.Contains(err.Error(), "addon not found") {
		t.Errorf("Error should mention addon not found: %v", err)
	}
}

func TestAddonEnableAlreadyEnabled(t *testing.T) {
	cleanup := setupTestRegistry(t, defaultTestAddons())
	defer cleanup()

	// Create temp directory for test
	tmpDir := t.TempDir()

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
		w.Close()
		os.Stdout = oldStdout
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
	// Disable doesn't need registry - it only modifies config
	tmpDir := t.TempDir()

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
		w.Close()
		os.Stdout = oldStdout
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
	// Disable doesn't need registry - it only modifies config
	tmpDir := t.TempDir()

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
		w.Close()
		os.Stdout = oldStdout
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
