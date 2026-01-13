package web

import (
	"html/template"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTemplateLoading verifies that all templates are loaded correctly
func TestTemplateLoading(t *testing.T) {
	// Load templates using the same method as server
	funcMap := template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
	}

	tmpl, err := loadAllTemplates(funcMap)
	if err != nil {
		t.Fatalf("failed to load templates: %v", err)
	}

	// List of templates that must exist (without templates/ prefix)
	requiredTemplates := []string{
		// Layouts
		"layouts/base.html",
		"layouts/wizard.html",
		// Setup pages
		"pages/setup/welcome.html",
		"pages/setup/domain.html",
		"pages/setup/admin.html",
		"pages/setup/storage.html",
		"pages/setup/vpn.html",
		"pages/setup/addons.html",
		"pages/setup/summary.html",
		"pages/setup/complete.html",
		// Dashboard pages
		"pages/dashboard.html",
		"pages/services.html",
		"pages/service_info.html",
		"pages/logs.html",
		"pages/addons.html",
		"pages/config.html",
		"pages/backup.html",
	}

	for _, name := range requiredTemplates {
		tmplFound := tmpl.Lookup(name)
		if tmplFound == nil {
			t.Errorf("template %q not found", name)
		}
	}
}

// TestEmbeddedFS verifies that the embedded filesystem contains expected files
func TestEmbeddedFS(t *testing.T) {
	// Check that templatesFS is accessible
	entries, err := fs.ReadDir(templatesFS, "templates")
	if err != nil {
		t.Fatalf("failed to read templates directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("templates directory is empty")
	}

	// Check for key subdirectories
	expectedDirs := []string{"layouts", "pages", "components"}
	for _, dir := range expectedDirs {
		_, err := fs.ReadDir(templatesFS, "templates/"+dir)
		if err != nil {
			t.Errorf("expected directory templates/%s not found: %v", dir, err)
		}
	}
}

// loadAllTemplates loads all templates from the embedded FS
// This mirrors the server's loadTemplates method
func loadAllTemplates(funcMap template.FuncMap) (*template.Template, error) {
	tmpl := template.New("").Funcs(funcMap)

	// Walk the embedded filesystem to find all .html files
	err := fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Only process .html files
		if strings.HasSuffix(path, ".html") {
			content, err := fs.ReadFile(templatesFS, path)
			if err != nil {
				return err
			}
			// Strip "templates/" prefix to match handler expectations
			templateName := strings.TrimPrefix(path, "templates/")
			_, err = tmpl.New(templateName).Parse(string(content))
			if err != nil {
				return err
			}
		}
		return nil
	})

	return tmpl, err
}

// TestGenerateSecureToken verifies secure token generation
func TestGenerateSecureToken(t *testing.T) {
	tests := []struct {
		bytes       int
		expectedLen int
	}{
		{16, 22}, // 16 bytes = 128 bits = ~22 base64 chars
		{32, 43}, // 32 bytes = 256 bits = ~43 base64 chars
		{64, 86}, // 64 bytes = 512 bits = ~86 base64 chars
	}

	for _, tt := range tests {
		token, err := generateSecureToken(tt.bytes)
		if err != nil {
			t.Errorf("generateSecureToken(%d) returned error: %v", tt.bytes, err)
			continue
		}

		if len(token) != tt.expectedLen {
			t.Errorf("generateSecureToken(%d) returned token of length %d, expected %d",
				tt.bytes, len(token), tt.expectedLen)
		}

		// Verify tokens are unique
		token2, _ := generateSecureToken(tt.bytes)
		if token == token2 {
			t.Errorf("generateSecureToken produced duplicate tokens")
		}
	}
}

// TestGetLocalIP verifies local IP detection
func TestGetLocalIP(t *testing.T) {
	ip := getLocalIP()

	// Should return something - either localhost or an IP
	if ip == "" {
		t.Error("getLocalIP returned empty string")
	}

	// Should not be empty
	if len(ip) == 0 {
		t.Error("getLocalIP returned empty IP")
	}
}

// TestServerConfig verifies server configuration
func TestServerConfig(t *testing.T) {
	cfg := &ServerConfig{
		Host:       "0.0.0.0",
		Port:       3000,
		ProjectDir: "/tmp/test",
	}

	server := NewServer(cfg)

	if server.config.Host != "0.0.0.0" {
		t.Errorf("expected host '0.0.0.0', got %q", server.config.Host)
	}

	if server.config.Port != 3000 {
		t.Errorf("expected port 3000, got %d", server.config.Port)
	}

	if server.config.ProjectDir != "/tmp/test" {
		t.Errorf("expected projectDir '/tmp/test', got %q", server.config.ProjectDir)
	}
}

// TestHealthEndpoint verifies the health endpoint
func TestHealthEndpoint(t *testing.T) {
	server := NewServer(&ServerConfig{
		Host:       "localhost",
		Port:       3000,
		ProjectDir: t.TempDir(),
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("expected body 'OK', got %q", w.Body.String())
	}
}

// TestStaticFSContents verifies static files are embedded
func TestStaticFSContents(t *testing.T) {
	// Check that staticFS is accessible
	entries, err := fs.ReadDir(staticFS, "static")
	if err != nil {
		t.Fatalf("failed to read static directory: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("static directory is empty")
	}

	// Check for key files/directories
	expectedDirs := []string{"css", "js"}
	for _, dir := range expectedDirs {
		_, err := fs.ReadDir(staticFS, "static/"+dir)
		if err != nil {
			t.Errorf("expected directory static/%s not found: %v", dir, err)
		}
	}
}

// TestPhaseDetectionPreInit verifies pre-init phase detection
func TestPhaseDetectionPreInit(t *testing.T) {
	// Create temp directory without config file
	tmpDir := t.TempDir()

	server := NewServer(&ServerConfig{
		Host:       "localhost",
		Port:       3000,
		ProjectDir: tmpDir,
	})

	err := server.checkPhase()
	if err != nil {
		t.Fatalf("checkPhase failed: %v", err)
	}

	if server.initialized {
		t.Error("expected initialized=false for pre-init phase")
	}

	if server.setupToken == "" {
		t.Error("expected setupToken to be generated for pre-init phase")
	}
}

// TestPhaseDetectionPostInit verifies post-init phase detection
func TestPhaseDetectionPostInit(t *testing.T) {
	// Create temp directory with config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".sdbx.yaml")
	if err := os.WriteFile(configPath, []byte("domain: test.local"), 0o644); err != nil {
		t.Fatalf("failed to create config file: %v", err)
	}

	server := NewServer(&ServerConfig{
		Host:       "localhost",
		Port:       3000,
		ProjectDir: tmpDir,
	})

	err := server.checkPhase()
	if err != nil {
		t.Fatalf("checkPhase failed: %v", err)
	}

	if !server.initialized {
		t.Error("expected initialized=true for post-init phase")
	}

	if server.setupToken != "" {
		t.Error("expected setupToken to be empty for post-init phase")
	}
}

// TestFormatServerMessagePreInit verifies pre-init startup message
func TestFormatServerMessagePreInit(t *testing.T) {
	server := NewServer(&ServerConfig{
		Host:       "0.0.0.0",
		Port:       3000,
		ProjectDir: t.TempDir(),
	})

	// Simulate pre-init state
	server.initialized = false
	server.setupToken = "test-token-123"

	msg := server.formatServerMessage()

	if !strings.Contains(msg, "Setup wizard") {
		t.Error("pre-init message should mention setup wizard")
	}

	if !strings.Contains(msg, "test-token-123") {
		t.Error("pre-init message should contain setup token")
	}
}

// TestFormatServerMessagePostInit verifies post-init startup message
func TestFormatServerMessagePostInit(t *testing.T) {
	server := NewServer(&ServerConfig{
		Host:       "0.0.0.0",
		Port:       3000,
		ProjectDir: t.TempDir(),
	})

	// Simulate post-init state
	server.initialized = true
	server.dockerMode = false

	msg := server.formatServerMessage()

	if !strings.Contains(msg, "development mode") {
		t.Error("post-init non-docker message should mention development mode")
	}

	// Test docker mode
	server.dockerMode = true
	msg = server.formatServerMessage()

	if !strings.Contains(msg, "production mode") {
		t.Error("post-init docker message should mention production mode")
	}
}

// TestTemplateFuncMap verifies template functions work
func TestTemplateFuncMap(t *testing.T) {
	funcMap := template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
	}

	// Test the sub function
	subFunc := funcMap["sub"].(func(int, int) int)
	result := subFunc(10, 3)
	if result != 7 {
		t.Errorf("sub(10, 3) = %d, expected 7", result)
	}
}
