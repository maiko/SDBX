package registry

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestNewLoader tests Loader creation
func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	if loader == nil {
		t.Fatal("NewLoader returned nil")
	}
}

// TestLoaderLoadServiceDefinition tests loading a service from file
func TestLoaderLoadServiceDefinition(t *testing.T) {
	tmpDir := t.TempDir()

	serviceYAML := `apiVersion: sdbx.io/v1
kind: Service
metadata:
  name: test-service
  version: 1.0.0
  category: utility
  description: Test service for testing
spec:
  image:
    repository: nginx
    tag: alpine
routing:
  enabled: true
  port: 80
`
	path := filepath.Join(tmpDir, "service.yaml")
	if err := os.WriteFile(path, []byte(serviceYAML), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loader := NewLoader()
	def, err := loader.LoadServiceDefinition(path)
	if err != nil {
		t.Fatalf("LoadServiceDefinition failed: %v", err)
	}

	if def.Metadata.Name != "test-service" {
		t.Errorf("name = %q, want 'test-service'", def.Metadata.Name)
	}

	if def.Spec.Image.Tag != "alpine" {
		t.Errorf("image tag = %q, want 'alpine'", def.Spec.Image.Tag)
	}
}

// TestLoaderLoadServiceDefinitionNotFound tests error for missing file
func TestLoaderLoadServiceDefinitionNotFound(t *testing.T) {
	loader := NewLoader()
	_, err := loader.LoadServiceDefinition("/nonexistent/path/service.yaml")

	if err == nil {
		t.Error("LoadServiceDefinition should return error for non-existent file")
	}
}

// TestLoaderParseServiceDefinition tests parsing YAML
func TestLoaderParseServiceDefinition(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid service",
			yaml: `apiVersion: sdbx.io/v1
kind: Service
metadata:
  name: valid-service
  version: 1.0.0
spec:
  image:
    repository: nginx
`,
			wantErr: false,
		},
		{
			name:    "invalid YAML",
			yaml:    "invalid: yaml: :::",
			wantErr: true,
			errMsg:  "failed to parse YAML",
		},
		{
			name: "wrong API version",
			yaml: `apiVersion: sdbx.io/v99
kind: Service
metadata:
  name: test
spec:
  image:
    repository: nginx
`,
			wantErr: true,
			errMsg:  "unsupported API version",
		},
		{
			name: "wrong kind",
			yaml: `apiVersion: sdbx.io/v1
kind: WrongKind
metadata:
  name: test
spec:
  image:
    repository: nginx
`,
			wantErr: true,
			errMsg:  "unexpected kind",
		},
	}

	loader := NewLoader()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := loader.ParseServiceDefinition([]byte(tt.yaml))

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" {
					if !bytes.Contains([]byte(err.Error()), []byte(tt.errMsg)) {
						t.Errorf("error = %q, want to contain %q", err.Error(), tt.errMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if def == nil {
					t.Error("expected definition, got nil")
				}
			}
		})
	}
}

// TestLoaderApplyDefaults tests default value application
func TestLoaderApplyDefaults(t *testing.T) {
	yaml := `apiVersion: sdbx.io/v1
kind: Service
metadata:
  name: defaults-test
  version: 1.0.0
spec:
  image:
    repository: nginx
routing:
  enabled: true
  port: 80
`
	loader := NewLoader()
	def, err := loader.ParseServiceDefinition([]byte(yaml))
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Check defaults were applied
	if def.Spec.Container.Restart != "unless-stopped" {
		t.Errorf("restart = %q, want 'unless-stopped'", def.Spec.Container.Restart)
	}

	if def.Spec.Container.NameTemplate != "sdbx-{{ .Name }}" {
		t.Errorf("name template = %q, want 'sdbx-{{ .Name }}'", def.Spec.Container.NameTemplate)
	}

	if def.Spec.Image.Registry != "docker.io" {
		t.Errorf("registry = %q, want 'docker.io'", def.Spec.Image.Registry)
	}

	if def.Spec.Image.Tag != "latest" {
		t.Errorf("tag = %q, want 'latest'", def.Spec.Image.Tag)
	}

	if def.Spec.Networking.Mode != "bridge" {
		t.Errorf("network mode = %q, want 'bridge'", def.Spec.Networking.Mode)
	}

	// Routing defaults
	if def.Routing.Subdomain != "defaults-test" {
		t.Errorf("subdomain = %q, want 'defaults-test'", def.Routing.Subdomain)
	}

	if def.Routing.Path != "/defaults-test" {
		t.Errorf("path = %q, want '/defaults-test'", def.Routing.Path)
	}

	if def.Routing.PathRouting.Strategy != "stripPrefix" {
		t.Errorf("path routing strategy = %q, want 'stripPrefix'", def.Routing.PathRouting.Strategy)
	}

	// Watchtower default
	if def.Integrations.Watchtower == nil || !def.Integrations.Watchtower.Enabled {
		t.Error("watchtower should be enabled by default")
	}
}

// TestLoaderLoadServiceOverride tests loading service overrides
func TestLoaderLoadServiceOverride(t *testing.T) {
	tmpDir := t.TempDir()

	overrideYAML := `apiVersion: sdbx.io/v1
kind: ServiceOverride
metadata:
  name: nginx
spec:
  image:
    tag: custom-tag
`
	path := filepath.Join(tmpDir, "override.yaml")
	if err := os.WriteFile(path, []byte(overrideYAML), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loader := NewLoader()
	override, err := loader.LoadServiceOverride(path)
	if err != nil {
		t.Fatalf("LoadServiceOverride failed: %v", err)
	}

	if override.Metadata.Name != "nginx" {
		t.Errorf("target name = %q, want 'nginx'", override.Metadata.Name)
	}
}

// TestLoaderParseServiceOverrideErrors tests override parsing errors
func TestLoaderParseServiceOverrideErrors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "invalid YAML",
			yaml:    "not: valid: yaml:::",
			wantErr: "failed to parse YAML",
		},
		{
			name: "wrong API version",
			yaml: `apiVersion: wrong
kind: ServiceOverride
target:
  name: test
`,
			wantErr: "unsupported API version",
		},
		{
			name: "wrong kind",
			yaml: `apiVersion: sdbx.io/v1
kind: WrongKind
metadata:
  name: test
`,
			wantErr: "unexpected kind",
		},
	}

	loader := NewLoader()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loader.ParseServiceOverride([]byte(tt.yaml))
			if err == nil {
				t.Error("expected error")
			} else if !bytes.Contains([]byte(err.Error()), []byte(tt.wantErr)) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestLoaderLoadSourceConfig tests loading source configuration
func TestLoaderLoadSourceConfig(t *testing.T) {
	tmpDir := t.TempDir()

	configYAML := `sources:
  - name: official
    type: git
    url: https://github.com/example/services
    priority: 0
    enabled: true
  - name: local
    type: local
    path: ~/.config/sdbx/services
    priority: 100
    enabled: true
`
	path := filepath.Join(tmpDir, "sources.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loader := NewLoader()
	cfg, err := loader.LoadSourceConfig(path)
	if err != nil {
		t.Fatalf("LoadSourceConfig failed: %v", err)
	}

	if len(cfg.Sources) != 2 {
		t.Errorf("sources count = %d, want 2", len(cfg.Sources))
	}

	// Check default branch was applied to git source
	if cfg.Sources[0].Branch != "main" {
		t.Errorf("default branch = %q, want 'main'", cfg.Sources[0].Branch)
	}
}

// TestLoaderLoadLockFile tests loading lock files
func TestLoaderLoadLockFile(t *testing.T) {
	tmpDir := t.TempDir()

	lockYAML := `apiVersion: sdbx.io/v1
kind: LockFile
metadata:
  version: 1
  cliVersion: "1.0.0"
  configHash: "sha256:abc123"
sources:
  official:
    url: https://github.com/example/services
    commit: abc123def456
    branch: main
services:
  nginx:
    source: official
    definitionVersion: "1.0.0"
    image:
      repository: nginx
      tag: latest
`
	path := filepath.Join(tmpDir, ".sdbx.lock")
	if err := os.WriteFile(path, []byte(lockYAML), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loader := NewLoader()
	lock, err := loader.LoadLockFile(path)
	if err != nil {
		t.Fatalf("LoadLockFile failed: %v", err)
	}

	if lock.Metadata.CLIVersion != "1.0.0" {
		t.Errorf("CLI version = %q, want '1.0.0'", lock.Metadata.CLIVersion)
	}

	if len(lock.Sources) != 1 {
		t.Errorf("sources count = %d, want 1", len(lock.Sources))
	}

	if lock.Sources["official"].Commit != "abc123def456" {
		t.Errorf("commit = %q, want 'abc123def456'", lock.Sources["official"].Commit)
	}
}

// TestLoaderParseLockFileErrors tests lock file parsing errors
func TestLoaderParseLockFileErrors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "invalid YAML",
			yaml:    "not: valid: ::yaml",
			wantErr: "failed to parse YAML",
		},
		{
			name: "wrong API version",
			yaml: `apiVersion: wrong
kind: LockFile
metadata:
  version: 1
`,
			wantErr: "unsupported API version",
		},
		{
			name: "wrong kind",
			yaml: `apiVersion: sdbx.io/v1
kind: WrongKind
metadata:
  version: 1
`,
			wantErr: "unexpected kind",
		},
	}

	loader := NewLoader()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loader.ParseLockFile([]byte(tt.yaml))
			if err == nil {
				t.Error("expected error")
			} else if !bytes.Contains([]byte(err.Error()), []byte(tt.wantErr)) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

// TestLoaderLoadSourceRepository tests loading source repository metadata
func TestLoaderLoadSourceRepository(t *testing.T) {
	tmpDir := t.TempDir()

	repoYAML := `apiVersion: sdbx.io/v1
kind: SourceRepository
metadata:
  name: official-services
  version: 1.0.0
  description: Official SDBX services repository
schemaVersion: "1.0"
`
	path := filepath.Join(tmpDir, "repository.yaml")
	if err := os.WriteFile(path, []byte(repoYAML), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	loader := NewLoader()
	repo, err := loader.LoadSourceRepository(path)
	if err != nil {
		t.Fatalf("LoadSourceRepository failed: %v", err)
	}

	if repo.Metadata.Name != "official-services" {
		t.Errorf("name = %q, want 'official-services'", repo.Metadata.Name)
	}

	if repo.Metadata.Version != "1.0.0" {
		t.Errorf("version = %q, want '1.0.0'", repo.Metadata.Version)
	}
}

// TestLoaderSaveServiceDefinition tests saving a service definition
func TestLoaderSaveServiceDefinition(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "subdir", "service.yaml")

	def := &ServiceDefinition{
		APIVersion: "sdbx.io/v1",
		Kind:       "Service",
		Metadata: ServiceMetadata{
			Name:        "saved-service",
			Version:     "1.0.0",
			Category:    "utility",
			Description: "A saved service",
		},
		Spec: ServiceSpec{
			Image: ImageSpec{
				Repository: "nginx",
				Tag:        "latest",
			},
		},
	}

	loader := NewLoader()
	if err := loader.SaveServiceDefinition(path, def); err != nil {
		t.Fatalf("SaveServiceDefinition failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("service.yaml should be created")
	}

	// Load it back
	loaded, err := loader.LoadServiceDefinition(path)
	if err != nil {
		t.Fatalf("failed to load saved definition: %v", err)
	}

	if loaded.Metadata.Name != "saved-service" {
		t.Errorf("loaded name = %q, want 'saved-service'", loaded.Metadata.Name)
	}
}

// TestLoaderSaveSourceConfig tests saving source configuration
func TestLoaderSaveSourceConfig(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "sources.yaml")

	cfg := &SourceConfig{
		Sources: []Source{
			{
				Name:     "test",
				Type:     "git",
				URL:      "https://github.com/test/repo",
				Branch:   "main",
				Priority: 0,
				Enabled:  true,
			},
		},
	}

	loader := NewLoader()
	if err := loader.SaveSourceConfig(path, cfg); err != nil {
		t.Fatalf("SaveSourceConfig failed: %v", err)
	}

	// Load it back
	loaded, err := loader.LoadSourceConfig(path)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if len(loaded.Sources) != 1 {
		t.Errorf("sources count = %d, want 1", len(loaded.Sources))
	}

	if loaded.Sources[0].Name != "test" {
		t.Errorf("source name = %q, want 'test'", loaded.Sources[0].Name)
	}
}

// TestLoaderSaveLockFile tests saving lock files
func TestLoaderSaveLockFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, ".sdbx.lock")

	lock := &LockFile{
		APIVersion: "sdbx.io/v1",
		Kind:       "LockFile",
		Metadata: LockFileMetadata{
			Version:    1,
			CLIVersion: "1.0.0",
			ConfigHash: "sha256:test",
		},
		Sources: map[string]LockedSource{
			"test": {
				URL:    "https://example.com",
				Commit: "abc123",
			},
		},
		Services: map[string]LockedService{
			"nginx": {
				Source:            "test",
				DefinitionVersion: "1.0.0",
			},
		},
	}

	loader := NewLoader()
	if err := loader.SaveLockFile(path, lock); err != nil {
		t.Fatalf("SaveLockFile failed: %v", err)
	}

	// Load it back
	loaded, err := loader.LoadLockFile(path)
	if err != nil {
		t.Fatalf("failed to load saved lock: %v", err)
	}

	if loaded.Metadata.CLIVersion != "1.0.0" {
		t.Errorf("CLI version = %q, want '1.0.0'", loaded.Metadata.CLIVersion)
	}
}

// TestLoaderDiscoverServices tests service discovery
func TestLoaderDiscoverServices(t *testing.T) {
	tmpDir := t.TempDir()

	// Create services in different locations
	services := []struct {
		path string
		name string
	}{
		{"service1", "service1"},
		{"core/service2", "service2"},
		{"addons/service3", "service3"},
	}

	for _, s := range services {
		dir := filepath.Join(tmpDir, s.path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "service.yaml"), []byte("test"), 0o644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
	}

	// Create hidden directory (should be skipped)
	hiddenDir := filepath.Join(tmpDir, ".hidden", "hidden-service")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatalf("failed to create hidden dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "service.yaml"), []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create hidden file: %v", err)
	}

	loader := NewLoader()
	discovered, err := loader.DiscoverServices(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverServices failed: %v", err)
	}

	// Should find 3 services, not the hidden one
	if len(discovered) != 3 {
		t.Errorf("discovered %d services, want 3", len(discovered))
	}

	// Verify hidden service not included
	for _, name := range discovered {
		if name == "hidden-service" {
			t.Error("hidden service should not be discovered")
		}
	}
}

// TestLoaderMergeOverride tests merging overrides
func TestLoaderMergeOverride(t *testing.T) {
	base := &ServiceDefinition{
		APIVersion: "sdbx.io/v1",
		Kind:       "Service",
		Metadata: ServiceMetadata{
			Name:    "base-service",
			Version: "1.0.0",
		},
		Spec: ServiceSpec{
			Image: ImageSpec{
				Repository: "nginx",
				Tag:        "latest",
				Registry:   "docker.io",
			},
			Environment: EnvironmentSpec{
				Static: []EnvVar{
					{Name: "EXISTING", Value: "value"},
				},
			},
			Volumes: []VolumeMount{
				{Name: "existing-vol", HostPath: "/host/existing", ContainerPath: "/existing"},
			},
		},
		Routing: RoutingConfig{
			Enabled:   true,
			Subdomain: "base",
			Path:      "/base",
		},
	}

	newSubdomain := "custom"
	newPath := "/custom"

	override := &ServiceOverride{
		APIVersion: "sdbx.io/v1",
		Kind:       "ServiceOverride",
		Metadata: OverrideMetadata{
			Name: "base-service",
		},
		Spec: &ServiceSpecOverride{
			Image: &ImageSpec{
				Tag: "custom",
			},
			Environment: &EnvironmentOverride{
				Additional: []EnvVar{
					{Name: "NEW_VAR", Value: "new_value"},
				},
			},
			Volumes: &VolumeOverride{
				Additional: []VolumeMount{
					{Name: "new-vol", HostPath: "/host/new", ContainerPath: "/new"},
				},
			},
		},
		Routing: &RoutingConfigOverride{
			Subdomain: &newSubdomain,
			Path:      &newPath,
		},
	}

	loader := NewLoader()
	merged := loader.MergeOverride(base, override)

	// Image should be overridden
	if merged.Spec.Image.Tag != "custom" {
		t.Errorf("image tag = %q, want 'custom'", merged.Spec.Image.Tag)
	}

	// Repository should stay the same
	if merged.Spec.Image.Repository != "nginx" {
		t.Errorf("repository = %q, want 'nginx'", merged.Spec.Image.Repository)
	}

	// Environment should be merged
	if len(merged.Spec.Environment.Static) != 2 {
		t.Errorf("env var count = %d, want 2", len(merged.Spec.Environment.Static))
	}

	// Volumes should be merged
	if len(merged.Spec.Volumes) != 2 {
		t.Errorf("volume count = %d, want 2", len(merged.Spec.Volumes))
	}

	// Routing should be overridden
	if merged.Routing.Subdomain != "custom" {
		t.Errorf("subdomain = %q, want 'custom'", merged.Routing.Subdomain)
	}

	if merged.Routing.Path != "/custom" {
		t.Errorf("path = %q, want '/custom'", merged.Routing.Path)
	}

	// Original should be unchanged
	if base.Spec.Image.Tag != "latest" {
		t.Error("base should not be modified")
	}
}

// TestWriteYAML tests YAML writing to a writer
func TestWriteYAML(t *testing.T) {
	var buf bytes.Buffer

	data := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	if err := WriteYAML(&buf, data); err != nil {
		t.Fatalf("WriteYAML failed: %v", err)
	}

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("key1")) {
		t.Error("output should contain 'key1'")
	}
	if !bytes.Contains([]byte(output), []byte("value1")) {
		t.Error("output should contain 'value1'")
	}
}
