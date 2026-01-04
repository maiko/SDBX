package registry

import (
	"errors"
	"testing"
)

// TestResolutionErrorError verifies ResolutionError.Error() method
func TestResolutionErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      ResolutionError
		expected string
	}{
		{
			name: "without cause",
			err: ResolutionError{
				Service: "test-service",
				Message: "service not found",
			},
			expected: "test-service: service not found",
		},
		{
			name: "with cause",
			err: ResolutionError{
				Service: "test-service",
				Message: "failed to load",
				Cause:   errors.New("file not found"),
			},
			expected: "test-service: failed to load: file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestValidationErrorError verifies ValidationError.Error() method
func TestValidationErrorError(t *testing.T) {
	err := ValidationError{
		Field:    "metadata.name",
		Message:  "name is required",
		Severity: "error",
	}

	expected := "metadata.name: name is required"
	if got := err.Error(); got != expected {
		t.Errorf("Error() = %q, want %q", got, expected)
	}
}

// TestServiceCategories verifies service category constants
func TestServiceCategories(t *testing.T) {
	categories := []struct {
		category ServiceCategory
		value    string
	}{
		{CategoryMedia, "media"},
		{CategoryDownloads, "downloads"},
		{CategoryManagement, "management"},
		{CategoryUtility, "utility"},
		{CategoryNetworking, "networking"},
		{CategoryAuth, "auth"},
	}

	for _, tc := range categories {
		if string(tc.category) != tc.value {
			t.Errorf("category %v should have value %q", tc.category, tc.value)
		}
	}
}

// TestAPIConstants verifies API version and kind constants
func TestAPIConstants(t *testing.T) {
	if APIVersion != "sdbx.io/v1" {
		t.Errorf("APIVersion = %q, want 'sdbx.io/v1'", APIVersion)
	}

	if KindService != "Service" {
		t.Errorf("KindService = %q, want 'Service'", KindService)
	}

	if KindServiceOverride != "ServiceOverride" {
		t.Errorf("KindServiceOverride = %q, want 'ServiceOverride'", KindServiceOverride)
	}

	if KindSourceRepository != "SourceRepository" {
		t.Errorf("KindSourceRepository = %q, want 'SourceRepository'", KindSourceRepository)
	}

	if KindSourceConfig != "SourceConfig" {
		t.Errorf("KindSourceConfig = %q, want 'SourceConfig'", KindSourceConfig)
	}

	if KindLockFile != "LockFile" {
		t.Errorf("KindLockFile = %q, want 'LockFile'", KindLockFile)
	}
}

// TestServiceDefinitionStruct verifies ServiceDefinition can be constructed
func TestServiceDefinitionStruct(t *testing.T) {
	def := &ServiceDefinition{
		APIVersion: APIVersion,
		Kind:       KindService,
		Metadata: ServiceMetadata{
			Name:        "test-service",
			Version:     "1.0.0",
			Category:    CategoryMedia,
			Description: "A test service",
			Homepage:    "https://example.com",
			Tags:        []string{"test", "example"},
		},
		Spec: ServiceSpec{
			Image: ImageSpec{
				Repository: "test/image",
				Tag:        "latest",
				Registry:   "docker.io",
			},
			Container: ContainerSpec{
				NameTemplate: "sdbx-{{ .Name }}",
				Restart:      "unless-stopped",
			},
			Environment: EnvironmentSpec{
				Static: []EnvVar{
					{Name: "TZ", Value: "UTC"},
				},
			},
			Volumes: []VolumeMount{
				{
					Name:          "config",
					HostPath:      "./configs/test",
					ContainerPath: "/config",
				},
			},
		},
		Routing: RoutingConfig{
			Enabled:   true,
			Port:      8080,
			Subdomain: "test",
			Auth: AuthConfig{
				Required: true,
			},
		},
		Conditions: Conditions{
			RequireAddon: true,
		},
		Integrations: Integrations{
			Homepage: &HomepageIntegration{
				Enabled: true,
				Group:   "Test",
			},
		},
	}

	if def.Metadata.Name != "test-service" {
		t.Errorf("Name = %q, want 'test-service'", def.Metadata.Name)
	}

	if def.Spec.Container.NameTemplate != "sdbx-{{ .Name }}" {
		t.Error("NameTemplate not set correctly")
	}

	if !def.Routing.Enabled {
		t.Error("Routing should be enabled")
	}

	if !def.Conditions.RequireAddon {
		t.Error("RequireAddon should be true")
	}
}

// TestResolvedService verifies ResolvedService struct
func TestResolvedService(t *testing.T) {
	def := &ServiceDefinition{
		Metadata: ServiceMetadata{Name: "test"},
	}

	resolved := &ResolvedService{
		Name:           "test",
		Source:         "embedded",
		SourcePath:     "/path/to/service.yaml",
		Definition:     def,
		DefinitionHash: "abc123",
		Dependencies:   []string{"dep1", "dep2"},
		Enabled:        true,
	}

	if resolved.Name != "test" {
		t.Errorf("Name = %q, want 'test'", resolved.Name)
	}

	if len(resolved.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(resolved.Dependencies))
	}

	if !resolved.Enabled {
		t.Error("Enabled should be true")
	}
}

// TestResolutionGraph verifies ResolutionGraph struct
func TestResolutionGraph(t *testing.T) {
	graph := &ResolutionGraph{
		Services: map[string]*ResolvedService{
			"service-a": {Name: "service-a", Enabled: true},
			"service-b": {Name: "service-b", Enabled: true},
		},
		Order: []string{"service-a", "service-b"},
		Errors: []ResolutionError{
			{Service: "service-c", Message: "not found"},
		},
	}

	if len(graph.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(graph.Services))
	}

	if len(graph.Order) != 2 {
		t.Errorf("expected 2 in order, got %d", len(graph.Order))
	}

	if len(graph.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(graph.Errors))
	}
}

// TestLockFile verifies LockFile struct
func TestLockFile(t *testing.T) {
	lock := &LockFile{
		APIVersion: APIVersion,
		Kind:       KindLockFile,
		Metadata: LockFileMetadata{
			Version:    1,
			CLIVersion: "1.0.0",
			ConfigHash: "def456",
		},
		Sources: map[string]LockedSource{
			"official": {
				URL:    "https://github.com/maiko/SDBX-Services",
				Commit: "abc123",
				Branch: "main",
			},
		},
		Services: map[string]LockedService{
			"sonarr": {
				Source:            "official",
				DefinitionVersion: "1.0.0",
				Image: LockedImage{
					Repository: "linuxserver/sonarr",
					Tag:        "latest",
				},
				Enabled: true,
			},
		},
		InstallOrder: []string{"traefik", "authelia", "sonarr"},
	}

	if lock.Kind != KindLockFile {
		t.Errorf("Kind = %q, want %q", lock.Kind, KindLockFile)
	}

	if len(lock.Sources) != 1 {
		t.Errorf("expected 1 source, got %d", len(lock.Sources))
	}

	if len(lock.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(lock.Services))
	}

	if lock.Services["sonarr"].Image.Repository != "linuxserver/sonarr" {
		t.Error("Service image not set correctly")
	}
}

// TestSourceConfig verifies SourceConfig struct
func TestSourceConfig(t *testing.T) {
	cfg := &SourceConfig{
		APIVersion: APIVersion,
		Kind:       KindSourceConfig,
		Metadata: SourceConfigMetadata{
			Version: 1,
		},
		Sources: []Source{
			{
				Name:     "official",
				Type:     "git",
				URL:      "https://github.com/maiko/SDBX-Services",
				Branch:   "main",
				Priority: 0,
				Enabled:  true,
			},
			{
				Name:     "local",
				Type:     "local",
				Path:     "~/.config/sdbx/services",
				Priority: 100,
				Enabled:  true,
			},
		},
		Cache: CacheConfig{
			Directory: "~/.cache/sdbx",
			TTL:       "24h",
		},
		Security: SecurityConfig{
			AllowUnverified:   false,
			RequireSignatures: false,
		},
	}

	if len(cfg.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(cfg.Sources))
	}

	if cfg.Sources[0].Type != "git" {
		t.Errorf("first source type = %q, want 'git'", cfg.Sources[0].Type)
	}

	if cfg.Sources[1].Type != "local" {
		t.Errorf("second source type = %q, want 'local'", cfg.Sources[1].Type)
	}
}

// TestTrustLevel verifies TrustLevel struct
func TestTrustLevel(t *testing.T) {
	trust := TrustLevel{
		AllowPrivileged:   false,
		AllowHostNetwork:  false,
		AllowCapabilities: []string{"NET_ADMIN", "SYS_TIME"},
		AllowedRegistries: []string{"docker.io", "ghcr.io"},
	}

	if trust.AllowPrivileged {
		t.Error("AllowPrivileged should be false")
	}

	if len(trust.AllowCapabilities) != 2 {
		t.Errorf("expected 2 capabilities, got %d", len(trust.AllowCapabilities))
	}

	if len(trust.AllowedRegistries) != 2 {
		t.Errorf("expected 2 registries, got %d", len(trust.AllowedRegistries))
	}
}

// TestHealthCheck verifies HealthCheck struct
func TestHealthCheck(t *testing.T) {
	hc := &HealthCheck{
		Test:        []string{"CMD", "curl", "-f", "http://localhost:8080/health"},
		Interval:    "30s",
		Timeout:     "10s",
		Retries:     3,
		StartPeriod: "60s",
	}

	if len(hc.Test) != 4 {
		t.Errorf("expected 4 test args, got %d", len(hc.Test))
	}

	if hc.Test[0] != "CMD" {
		t.Errorf("first test arg = %q, want 'CMD'", hc.Test[0])
	}

	if hc.Retries != 3 {
		t.Errorf("Retries = %d, want 3", hc.Retries)
	}
}

// TestIntegrations verifies Integrations struct
func TestIntegrations(t *testing.T) {
	integrations := Integrations{
		Homepage: &HomepageIntegration{
			Enabled:     true,
			Group:       "Media",
			Icon:        "sonarr.png",
			Description: "TV Shows",
			Widget: &HomepageWidget{
				Type: "sonarr",
				Fields: map[string]string{
					"url": "{{HOMEPAGE_VAR_URL}}",
					"key": "{{HOMEPAGE_VAR_KEY}}",
				},
			},
		},
		Cloudflared: &CloudflaredIntegration{
			Enabled: true,
		},
		Watchtower: &WatchtowerIntegration{
			Enabled: true,
		},
		Unpackerr: &UnpackerrIntegration{
			Enabled:      true,
			URLEnvVar:    "UN_SONARR_0_URL",
			APIKeyEnvVar: "UN_SONARR_0_API_KEY",
			InternalURL:  "http://sdbx-sonarr:8989",
		},
	}

	if !integrations.Homepage.Enabled {
		t.Error("Homepage integration should be enabled")
	}

	if integrations.Homepage.Widget.Type != "sonarr" {
		t.Errorf("Widget type = %q, want 'sonarr'", integrations.Homepage.Widget.Type)
	}

	if !integrations.Cloudflared.Enabled {
		t.Error("Cloudflared integration should be enabled")
	}

	if !integrations.Watchtower.Enabled {
		t.Error("Watchtower integration should be enabled")
	}

	if integrations.Unpackerr.InternalURL != "http://sdbx-sonarr:8989" {
		t.Error("Unpackerr internal URL not set correctly")
	}
}
