package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maiko/sdbx/internal/config"
)

func TestNewGenerator(t *testing.T) {
	cfg := config.DefaultConfig()
	tmpDir := "/tmp/test"

	gen := NewGenerator(cfg, tmpDir)

	if gen.Config != cfg {
		t.Error("Generator config should match input config")
	}
	if gen.OutputDir != tmpDir {
		t.Errorf("Generator OutputDir = %s, want %s", gen.OutputDir, tmpDir)
	}
}

func TestGenerateDirectoryStructure(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-gen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create generator
	cfg := config.DefaultConfig()
	gen := NewGenerator(cfg, tmpDir)

	// Generate project
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify directories were created
	expectedDirs := []string{
		"configs",
		"configs/traefik",
		"configs/traefik/dynamic",
		"configs/authelia",
		"configs/gluetun",
		"configs/homepage",
		"secrets",
	}

	for _, dir := range expectedDirs {
		path := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Directory %s should have been created", dir)
		}
	}
}

func TestGenerateWithCloudflared(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-gen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create generator with cloudflared mode
	cfg := config.DefaultConfig()
	cfg.Expose.Mode = "cloudflared"
	gen := NewGenerator(cfg, tmpDir)

	// Generate project
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify cloudflared config directory was created
	cloudflaredDir := filepath.Join(tmpDir, "configs/cloudflared")
	if _, err := os.Stat(cloudflaredDir); os.IsNotExist(err) {
		t.Error("Cloudflared config directory should have been created")
	}

	// Verify cloudflared config file was generated
	cloudflaredConfig := filepath.Join(tmpDir, "configs/cloudflared/config.yml")
	if _, err := os.Stat(cloudflaredConfig); os.IsNotExist(err) {
		t.Error("Cloudflared config file should have been generated")
	}
}

func TestGenerateWithoutCloudflared(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-gen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create generator with LAN mode (no cloudflared)
	cfg := config.DefaultConfig()
	cfg.Expose.Mode = "lan"
	gen := NewGenerator(cfg, tmpDir)

	// Generate project
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify cloudflared config directory was NOT created
	cloudflaredDir := filepath.Join(tmpDir, "configs/cloudflared")
	if _, err := os.Stat(cloudflaredDir); !os.IsNotExist(err) {
		t.Error("Cloudflared config directory should not exist in LAN mode")
	}
}

func TestGenerateFiles(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-gen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create generator
	cfg := config.DefaultConfig()
	gen := NewGenerator(cfg, tmpDir)

	// Generate project
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify core files were created
	expectedFiles := []string{
		"compose.yaml",
		".env",
		".sdbx.yaml",
		".gitignore",
		"configs/traefik/traefik.yml",
		"configs/traefik/dynamic/middlewares.yml",
		"configs/authelia/configuration.yml",
		"configs/authelia/users_database.yml",
		"configs/homepage/settings.yaml",
		"configs/homepage/services.yaml",
		"configs/homepage/docker.yaml",
		"configs/gluetun/gluetun.env",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("File %s should have been created", file)
		}
	}
}

func TestGenerateSecrets(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-gen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create generator
	cfg := config.DefaultConfig()
	gen := NewGenerator(cfg, tmpDir)

	// Generate project
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify secrets directory was created
	secretsDir := filepath.Join(tmpDir, "secrets")
	if _, err := os.Stat(secretsDir); os.IsNotExist(err) {
		t.Error("Secrets directory should have been created")
	}

	// Verify some secret files were created
	entries, err := os.ReadDir(secretsDir)
	if err != nil {
		t.Fatalf("Failed to read secrets dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("Secrets directory should contain files")
	}
}

func TestCreateDataDirs(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-gen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create generator with relative paths
	cfg := config.DefaultConfig()
	cfg.MediaPath = filepath.Join(tmpDir, "media")
	cfg.DownloadsPath = filepath.Join(tmpDir, "downloads")
	gen := NewGenerator(cfg, tmpDir)

	// Create data directories
	if err := gen.CreateDataDirs(); err != nil {
		t.Fatalf("CreateDataDirs failed: %v", err)
	}

	// Verify directories were created
	expectedDirs := []string{
		filepath.Join(tmpDir, "downloads"),
		filepath.Join(tmpDir, "media/movies"),
		filepath.Join(tmpDir, "media/tv"),
		filepath.Join(tmpDir, "media/music"),
		filepath.Join(tmpDir, "media/books"),
	}

	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s should have been created", dir)
		}
	}
}

func TestGenerateWithAddons(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-gen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create generator with addons enabled
	cfg := config.DefaultConfig()
	cfg.Addons = []string{"overseerr", "tautulli"}
	gen := NewGenerator(cfg, tmpDir)

	// Generate project
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Verify config directories for addons were created
	overseerrDir := filepath.Join(tmpDir, "configs/overseerr")
	if _, err := os.Stat(overseerrDir); os.IsNotExist(err) {
		t.Error("Overseerr config directory should have been created")
	}

	tautulliDir := filepath.Join(tmpDir, "configs/tautulli")
	if _, err := os.Stat(tautulliDir); os.IsNotExist(err) {
		t.Error("Tautulli config directory should have been created")
	}
}

func TestGenerateFileContent(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "sdbx-gen-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create generator with test values
	cfg := config.DefaultConfig()
	cfg.Domain = "test.example.com"
	cfg.Timezone = "America/New_York"
	gen := NewGenerator(cfg, tmpDir)

	// Generate project
	if err := gen.Generate(); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Read .env file and verify it contains config values
	envFile := filepath.Join(tmpDir, ".env")
	content, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("Failed to read .env file: %v", err)
	}

	envContent := string(content)
	if len(envContent) == 0 {
		t.Error(".env file should not be empty")
	}

	// Read compose.yaml and verify it exists and has content
	composeFile := filepath.Join(tmpDir, "compose.yaml")
	composeContent, err := os.ReadFile(composeFile)
	if err != nil {
		t.Fatalf("Failed to read compose.yaml: %v", err)
	}

	if len(composeContent) == 0 {
		t.Error("compose.yaml file should not be empty")
	}
}
