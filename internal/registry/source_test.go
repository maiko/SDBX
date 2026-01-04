package registry

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestNewLocalSource tests LocalSource creation
func TestNewLocalSource(t *testing.T) {
	src := NewLocalSource(Source{
		Name:     "test-local",
		Type:     "local",
		Priority: 100,
		Enabled:  true,
		Path:     "/tmp/test-services",
	})

	if src == nil {
		t.Fatal("NewLocalSource returned nil")
	}

	if src.Name() != "test-local" {
		t.Errorf("Name = %q, want 'test-local'", src.Name())
	}

	if src.Type() != "local" {
		t.Errorf("Type = %q, want 'local'", src.Type())
	}

	if src.Priority() != 100 {
		t.Errorf("Priority = %d, want 100", src.Priority())
	}

	if !src.IsEnabled() {
		t.Error("IsEnabled should return true")
	}
}

// TestNewLocalSourceDefaultPath tests default path
func TestNewLocalSourceDefaultPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		// No path provided
	})

	expectedPath := filepath.Join(home, ".config", "sdbx", "services")
	if src.GetPath() != expectedPath {
		t.Errorf("default path = %q, want %q", src.GetPath(), expectedPath)
	}
}

// TestNewLocalSourceTildeExpansion tests ~ expansion
func TestNewLocalSourceTildeExpansion(t *testing.T) {
	home, _ := os.UserHomeDir()
	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    "~/my-services",
	})

	expectedPath := filepath.Join(home, "my-services")
	if src.GetPath() != expectedPath {
		t.Errorf("tilde-expanded path = %q, want %q", src.GetPath(), expectedPath)
	}
}

// TestLocalSourceLoad tests loading services from directory
func TestLocalSourceLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a service definition
	serviceDir := filepath.Join(tmpDir, "test-service")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("failed to create service dir: %v", err)
	}

	serviceYAML := `apiVersion: sdbx.io/v1
kind: Service
metadata:
  name: test-service
  version: 1.0.0
  category: utility
  description: Test service
spec:
  image:
    repository: nginx
    tag: latest
`
	if err := os.WriteFile(filepath.Join(serviceDir, "service.yaml"), []byte(serviceYAML), 0o644); err != nil {
		t.Fatalf("failed to write service.yaml: %v", err)
	}

	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    tmpDir,
	})

	ctx := context.Background()
	services, err := src.Load(ctx)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(services) != 1 {
		t.Errorf("loaded %d services, want 1", len(services))
	}

	if services[0].Metadata.Name != "test-service" {
		t.Errorf("service name = %q, want 'test-service'", services[0].Metadata.Name)
	}
}

// TestLocalSourceLoadNonExistent tests loading from non-existent directory
func TestLocalSourceLoadNonExistent(t *testing.T) {
	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    "/nonexistent/path/that/doesnt/exist",
	})

	ctx := context.Background()
	services, err := src.Load(ctx)

	// Should return nil, nil for non-existent directory
	if err != nil {
		t.Errorf("Load returned error for non-existent: %v", err)
	}
	if services != nil {
		t.Error("services should be nil for non-existent directory")
	}
}

// TestLocalSourceLoadService tests loading a specific service
func TestLocalSourceLoadService(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a service in core/ subdirectory
	coreDir := filepath.Join(tmpDir, "core", "my-service")
	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatalf("failed to create core dir: %v", err)
	}

	serviceYAML := `apiVersion: sdbx.io/v1
kind: Service
metadata:
  name: my-service
  version: 1.0.0
  category: utility
  description: My service
spec:
  image:
    repository: nginx
    tag: alpine
`
	if err := os.WriteFile(filepath.Join(coreDir, "service.yaml"), []byte(serviceYAML), 0o644); err != nil {
		t.Fatalf("failed to write service.yaml: %v", err)
	}

	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    tmpDir,
	})

	ctx := context.Background()
	def, err := src.LoadService(ctx, "my-service")
	if err != nil {
		t.Fatalf("LoadService failed: %v", err)
	}

	if def.Metadata.Name != "my-service" {
		t.Errorf("service name = %q, want 'my-service'", def.Metadata.Name)
	}

	if def.Spec.Image.Tag != "alpine" {
		t.Errorf("image tag = %q, want 'alpine'", def.Spec.Image.Tag)
	}
}

// TestLocalSourceLoadServiceFromAddons tests loading from addons/ subdirectory
func TestLocalSourceLoadServiceFromAddons(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a service in addons/ subdirectory
	addonsDir := filepath.Join(tmpDir, "addons", "sonarr")
	if err := os.MkdirAll(addonsDir, 0o755); err != nil {
		t.Fatalf("failed to create addons dir: %v", err)
	}

	serviceYAML := `apiVersion: sdbx.io/v1
kind: Service
metadata:
  name: sonarr
  version: 1.0.0
  category: media
  description: TV automation
spec:
  image:
    repository: linuxserver/sonarr
    tag: latest
`
	if err := os.WriteFile(filepath.Join(addonsDir, "service.yaml"), []byte(serviceYAML), 0o644); err != nil {
		t.Fatalf("failed to write service.yaml: %v", err)
	}

	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    tmpDir,
	})

	ctx := context.Background()
	def, err := src.LoadService(ctx, "sonarr")
	if err != nil {
		t.Fatalf("LoadService failed: %v", err)
	}

	if def.Metadata.Category != "media" {
		t.Errorf("category = %q, want 'media'", def.Metadata.Category)
	}
}

// TestLocalSourceLoadServiceNotFound tests error for missing service
func TestLocalSourceLoadServiceNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    tmpDir,
	})

	ctx := context.Background()
	_, err := src.LoadService(ctx, "nonexistent-service")

	if err == nil {
		t.Error("LoadService should return error for non-existent service")
	}
}

// TestLocalSourceListServices tests service discovery
func TestLocalSourceListServices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple services
	services := []string{"service1", "service2", "service3"}
	for _, name := range services {
		serviceDir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(serviceDir, 0o755); err != nil {
			t.Fatalf("failed to create service dir: %v", err)
		}

		serviceYAML := `apiVersion: sdbx.io/v1
kind: Service
metadata:
  name: ` + name + `
  version: 1.0.0
spec:
  image:
    repository: nginx
    tag: latest
`
		if err := os.WriteFile(filepath.Join(serviceDir, "service.yaml"), []byte(serviceYAML), 0o644); err != nil {
			t.Fatalf("failed to write service.yaml: %v", err)
		}
	}

	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    tmpDir,
	})

	ctx := context.Background()
	listed, err := src.ListServices(ctx)
	if err != nil {
		t.Fatalf("ListServices failed: %v", err)
	}

	if len(listed) != 3 {
		t.Errorf("listed %d services, want 3", len(listed))
	}
}

// TestLocalSourceGetServicePath tests path resolution
func TestLocalSourceGetServicePath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create service in addons/
	addonsDir := filepath.Join(tmpDir, "addons", "test-addon")
	if err := os.MkdirAll(addonsDir, 0o755); err != nil {
		t.Fatalf("failed to create addons dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(addonsDir, "service.yaml"), []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to write service.yaml: %v", err)
	}

	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    tmpDir,
	})

	path := src.GetServicePath("test-addon")
	expectedPath := filepath.Join(tmpDir, "addons", "test-addon", "service.yaml")

	if path != expectedPath {
		t.Errorf("GetServicePath = %q, want %q", path, expectedPath)
	}
}

// TestLocalSourceUpdate tests update (no-op for local)
func TestLocalSourceUpdate(t *testing.T) {
	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    "/tmp/test",
	})

	ctx := context.Background()
	err := src.Update(ctx)

	// Should be a no-op, return nil
	if err != nil {
		t.Errorf("Update returned error: %v", err)
	}
}

// TestLocalSourceGetCommit tests commit (always empty for local)
func TestLocalSourceGetCommit(t *testing.T) {
	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    "/tmp/test",
	})

	commit := src.GetCommit()

	if commit != "" {
		t.Errorf("GetCommit = %q, want empty string", commit)
	}
}

// TestLocalSourceHasService tests service existence checking
func TestLocalSourceHasService(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a service
	serviceDir := filepath.Join(tmpDir, "existing-service")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("failed to create service dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "service.yaml"), []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to write service.yaml: %v", err)
	}

	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    tmpDir,
	})

	if !src.HasService("existing-service") {
		t.Error("HasService should return true for existing service")
	}

	if src.HasService("nonexistent-service") {
		t.Error("HasService should return false for non-existent service")
	}
}

// TestLocalSourceCreateServiceDir tests directory creation
func TestLocalSourceCreateServiceDir(t *testing.T) {
	tmpDir := t.TempDir()

	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    tmpDir,
	})

	// Create addon directory
	addonPath, err := src.CreateServiceDir("new-addon", true)
	if err != nil {
		t.Fatalf("CreateServiceDir failed: %v", err)
	}

	expectedAddonPath := filepath.Join(tmpDir, "addons", "new-addon")
	if addonPath != expectedAddonPath {
		t.Errorf("addon path = %q, want %q", addonPath, expectedAddonPath)
	}

	if _, err := os.Stat(addonPath); os.IsNotExist(err) {
		t.Error("addon directory should be created")
	}

	// Create core directory
	corePath, err := src.CreateServiceDir("new-core", false)
	if err != nil {
		t.Fatalf("CreateServiceDir failed: %v", err)
	}

	expectedCorePath := filepath.Join(tmpDir, "core", "new-core")
	if corePath != expectedCorePath {
		t.Errorf("core path = %q, want %q", corePath, expectedCorePath)
	}
}

// TestLocalSourceDeleteService tests service deletion
func TestLocalSourceDeleteService(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a service to delete
	serviceDir := filepath.Join(tmpDir, "to-delete")
	if err := os.MkdirAll(serviceDir, 0o755); err != nil {
		t.Fatalf("failed to create service dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(serviceDir, "service.yaml"), []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to write service.yaml: %v", err)
	}

	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    tmpDir,
	})

	// Delete
	if err := src.DeleteService("to-delete"); err != nil {
		t.Fatalf("DeleteService failed: %v", err)
	}

	// Verify deleted
	if _, err := os.Stat(serviceDir); !os.IsNotExist(err) {
		t.Error("service directory should be deleted")
	}
}

// TestLocalSourceDeleteServiceNotFound tests error for missing service
func TestLocalSourceDeleteServiceNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    tmpDir,
	})

	err := src.DeleteService("nonexistent")

	if err == nil {
		t.Error("DeleteService should return error for non-existent service")
	}
}

// TestBaseSourceMethods tests BaseSource embedded methods
func TestBaseSourceMethods(t *testing.T) {
	base := BaseSource{
		name:     "test",
		srcType:  "git",
		priority: 50,
		enabled:  true,
		path:     "/test/path",
	}

	if base.Name() != "test" {
		t.Errorf("Name = %q, want 'test'", base.Name())
	}

	if base.Type() != "git" {
		t.Errorf("Type = %q, want 'git'", base.Type())
	}

	if base.Priority() != 50 {
		t.Errorf("Priority = %d, want 50", base.Priority())
	}

	if !base.IsEnabled() {
		t.Error("IsEnabled should return true")
	}
}

// TestLocalSourceSaveService tests saving a service definition
func TestLocalSourceSaveService(t *testing.T) {
	tmpDir := t.TempDir()

	src := NewLocalSource(Source{
		Name:    "test-local",
		Enabled: true,
		Path:    tmpDir,
	})

	def := &ServiceDefinition{
		APIVersion: "sdbx.io/v1",
		Kind:       "Service",
		Metadata: ServiceMetadata{
			Name:        "new-service",
			Version:     "1.0.0",
			Category:    "utility",
			Description: "A new service",
		},
		Spec: ServiceSpec{
			Image: ImageSpec{
				Repository: "nginx",
				Tag:        "latest",
			},
		},
		Conditions: Conditions{
			RequireAddon: true, // This will place it in addons/
		},
	}

	if err := src.SaveService(def); err != nil {
		t.Fatalf("SaveService failed: %v", err)
	}

	// Verify file was created
	expectedPath := filepath.Join(tmpDir, "addons", "new-service", "service.yaml")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("service.yaml should be created")
	}

	// Load it back and verify
	ctx := context.Background()
	loaded, err := src.LoadService(ctx, "new-service")
	if err != nil {
		t.Fatalf("LoadService failed: %v", err)
	}

	if loaded.Metadata.Name != "new-service" {
		t.Errorf("loaded name = %q, want 'new-service'", loaded.Metadata.Name)
	}
}
