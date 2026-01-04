package registry

import (
	"testing"
)

// TestNewValidator verifies validator construction
func TestNewValidator(t *testing.T) {
	v := NewValidator()

	if v == nil {
		t.Fatal("NewValidator returned nil")
	}

	if len(v.allowedRegistries) == 0 {
		t.Error("allowedRegistries should be populated")
	}

	if len(v.dangerousCaps) == 0 {
		t.Error("dangerousCaps should be populated")
	}
}

// TestValidatorAllowedRegistries verifies allowed registries
func TestValidatorAllowedRegistries(t *testing.T) {
	v := NewValidator()

	expectedRegistries := []string{"docker.io", "ghcr.io", "lscr.io", "quay.io", "gcr.io", "registry.k8s.io"}
	for _, reg := range expectedRegistries {
		if !v.allowedRegistries[reg] {
			t.Errorf("expected registry %s to be allowed", reg)
		}
	}
}

// TestValidateMetadata verifies metadata validation
func TestValidateMetadata(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		def       *ServiceDefinition
		wantError bool
		field     string
	}{
		{
			name: "valid metadata",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:        "test-service",
					Version:     "1.0.0",
					Category:    CategoryMedia,
					Description: "A test service",
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
			},
			wantError: false,
		},
		{
			name: "missing name",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
			},
			wantError: true,
			field:     "metadata.name",
		},
		{
			name: "invalid name format",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "Invalid_Name",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
			},
			wantError: true,
			field:     "metadata.name",
		},
		{
			name: "missing version",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
			},
			wantError: true,
			field:     "metadata.version",
		},
		{
			name: "missing category",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:    "test",
					Version: "1.0.0",
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
			},
			wantError: true,
			field:     "metadata.category",
		},
		{
			name: "invalid category",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: "invalid-category",
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
			},
			wantError: true,
			field:     "metadata.category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := v.Validate(tt.def)
			hasError := HasErrors(errors)

			if hasError != tt.wantError {
				t.Errorf("expected error=%v, got error=%v, errors=%v", tt.wantError, hasError, errors)
			}

			if tt.wantError && tt.field != "" {
				found := false
				for _, e := range errors {
					if e.Field == tt.field && e.Severity == "error" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %s, got %v", tt.field, errors)
				}
			}
		})
	}
}

// TestValidateSpec verifies spec validation
func TestValidateSpec(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		def       *ServiceDefinition
		wantError bool
		field     string
	}{
		{
			name: "missing image repository",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: ""},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
			},
			wantError: true,
			field:     "spec.image.repository",
		},
		{
			name: "missing container name template",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: ""},
				},
			},
			wantError: true,
			field:     "spec.container.name_template",
		},
		{
			name: "volume missing host path",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
					Volumes: []VolumeMount{
						{HostPath: "", ContainerPath: "/data"},
					},
				},
			},
			wantError: true,
			field:     "spec.volumes[0].hostPath",
		},
		{
			name: "volume missing container path",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
					Volumes: []VolumeMount{
						{HostPath: "/host/data", ContainerPath: ""},
					},
				},
			},
			wantError: true,
			field:     "spec.volumes[0].containerPath",
		},
		{
			name: "env var missing name",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
					Environment: EnvironmentSpec{
						Static: []EnvVar{{Name: "", Value: "value"}},
					},
				},
			},
			wantError: true,
			field:     "spec.environment.static[0].name",
		},
		{
			name: "env var missing value and valueFrom",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
					Environment: EnvironmentSpec{
						Static: []EnvVar{{Name: "TEST"}},
					},
				},
			},
			wantError: true,
			field:     "spec.environment.static[0]",
		},
		{
			name: "conditional env missing when",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
					Environment: EnvironmentSpec{
						Conditional: []ConditionalEnvVar{
							{EnvVar: EnvVar{Name: "TEST", Value: "val"}, When: ""},
						},
					},
				},
			},
			wantError: true,
			field:     "spec.environment.conditional[0].when",
		},
		{
			name: "health check missing test",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:       ImageSpec{Repository: "test/image"},
					Container:   ContainerSpec{NameTemplate: "{{ .Name }}"},
					HealthCheck: &HealthCheck{Test: []string{}},
				},
			},
			wantError: true,
			field:     "spec.healthcheck.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := v.Validate(tt.def)
			hasError := HasErrors(errors)

			if hasError != tt.wantError {
				t.Errorf("expected error=%v, got error=%v, errors=%v", tt.wantError, hasError, errors)
			}

			if tt.wantError && tt.field != "" {
				found := false
				for _, e := range errors {
					if e.Field == tt.field && e.Severity == "error" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %s, got %v", tt.field, errors)
				}
			}
		})
	}
}

// TestValidateRouting verifies routing validation
func TestValidateRouting(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name      string
		def       *ServiceDefinition
		wantError bool
		field     string
	}{
		{
			name: "routing disabled - no validation needed",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
				Routing: RoutingConfig{Enabled: false},
			},
			wantError: false,
		},
		{
			name: "valid routing",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
				Routing: RoutingConfig{Enabled: true, Port: 8080},
			},
			wantError: false,
		},
		{
			name: "invalid port - zero",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
				Routing: RoutingConfig{Enabled: true, Port: 0},
			},
			wantError: true,
			field:     "routing.port",
		},
		{
			name: "invalid port - too high",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
				Routing: RoutingConfig{Enabled: true, Port: 70000},
			},
			wantError: true,
			field:     "routing.port",
		},
		{
			name: "invalid subdomain",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
				Routing: RoutingConfig{Enabled: true, Port: 8080, Subdomain: "Invalid_Sub"},
			},
			wantError: true,
			field:     "routing.subdomain",
		},
		{
			name: "path not starting with slash",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
				Routing: RoutingConfig{Enabled: true, Port: 8080, Path: "no-slash"},
			},
			wantError: true,
			field:     "routing.path",
		},
		{
			name: "invalid path routing strategy",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
				Routing: RoutingConfig{
					Enabled: true,
					Port:    8080,
					PathRouting: PathRoutingConfig{
						Strategy: "invalidStrategy",
					},
				},
			},
			wantError: true,
			field:     "routing.pathRouting.strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := v.Validate(tt.def)
			hasError := HasErrors(errors)

			if hasError != tt.wantError {
				t.Errorf("expected error=%v, got error=%v, errors=%v", tt.wantError, hasError, errors)
			}

			if tt.wantError && tt.field != "" {
				found := false
				for _, e := range errors {
					if e.Field == tt.field && e.Severity == "error" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error for field %s, got %v", tt.field, errors)
				}
			}
		})
	}
}

// TestValidateSecurity verifies security validation
func TestValidateSecurity(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name        string
		def         *ServiceDefinition
		wantError   bool
		wantWarning bool
		field       string
	}{
		{
			name: "privileged mode",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}", Privileged: true},
				},
			},
			wantError: true,
			field:     "spec.container.privileged",
		},
		{
			name: "dangerous capability",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image: ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{
						NameTemplate: "{{ .Name }}",
						Capabilities: CapabilitiesSpec{Add: []string{"SYS_ADMIN"}},
					},
				},
			},
			wantWarning: true,
			field:       "spec.container.capabilities.add",
		},
		{
			name: "host network mode",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:      ImageSpec{Repository: "test/image"},
					Container:  ContainerSpec{NameTemplate: "{{ .Name }}"},
					Networking: NetworkSpec{Mode: "host"},
				},
			},
			wantWarning: true,
			field:       "spec.networking.mode",
		},
		{
			name: "unallowed registry",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image", Registry: "untrusted.io"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
				},
			},
			wantWarning: true,
			field:       "spec.image.registry",
		},
		{
			name: "dangerous device mapping",
			def: &ServiceDefinition{
				Metadata: ServiceMetadata{
					Name:     "test",
					Version:  "1.0.0",
					Category: CategoryMedia,
				},
				Spec: ServiceSpec{
					Image:     ImageSpec{Repository: "test/image"},
					Container: ContainerSpec{NameTemplate: "{{ .Name }}", Devices: []string{"/dev/mem"}},
				},
			},
			wantError: true,
			field:     "spec.container.devices",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := v.Validate(tt.def)
			hasError := HasErrors(errors)
			warnings := FilterBySeverity(errors, "warning")
			hasWarning := len(warnings) > 0

			if tt.wantError && !hasError {
				t.Errorf("expected error, got none. errors=%v", errors)
			}

			if tt.wantWarning && !hasWarning {
				t.Errorf("expected warning, got none. errors=%v", errors)
			}
		})
	}
}

// TestValidateWithTrustLevel verifies trust level validation
func TestValidateWithTrustLevel(t *testing.T) {
	v := NewValidator()

	// Test that TrustLevel adds additional restrictions on top of base validation
	// Note: Base validation already flags privileged mode as error and host network as warning
	// TrustLevel restrictions are additive checks

	t.Run("privileged not allowed by trust level adds extra error", func(t *testing.T) {
		def := &ServiceDefinition{
			Metadata: ServiceMetadata{
				Name:     "test",
				Version:  "1.0.0",
				Category: CategoryMedia,
			},
			Spec: ServiceSpec{
				Image:     ImageSpec{Repository: "test/image"},
				Container: ContainerSpec{NameTemplate: "{{ .Name }}", Privileged: true},
			},
		}

		// With AllowPrivileged=false, should add trust-level specific error
		trust := TrustLevel{AllowPrivileged: false}
		errors := v.ValidateWithTrustLevel(def, trust)

		// Should have both base validation error AND trust level error for privileged
		privilegedErrors := 0
		for _, e := range errors {
			if e.Field == "spec.container.privileged" && e.Severity == "error" {
				privilegedErrors++
			}
		}
		if privilegedErrors < 2 {
			t.Errorf("expected at least 2 privileged errors (base + trust), got %d: %v", privilegedErrors, errors)
		}
	})

	t.Run("privileged allowed by trust level still has base validation error", func(t *testing.T) {
		def := &ServiceDefinition{
			Metadata: ServiceMetadata{
				Name:     "test",
				Version:  "1.0.0",
				Category: CategoryMedia,
			},
			Spec: ServiceSpec{
				Image:     ImageSpec{Repository: "test/image"},
				Container: ContainerSpec{NameTemplate: "{{ .Name }}", Privileged: true},
			},
		}

		// With AllowPrivileged=true, should only have base validation error
		trust := TrustLevel{AllowPrivileged: true}
		errors := v.ValidateWithTrustLevel(def, trust)

		// Should still have base validation error for privileged
		privilegedErrors := 0
		for _, e := range errors {
			if e.Field == "spec.container.privileged" && e.Severity == "error" {
				privilegedErrors++
			}
		}
		if privilegedErrors != 1 {
			t.Errorf("expected 1 privileged error (base only), got %d: %v", privilegedErrors, errors)
		}
	})

	t.Run("host network not allowed by trust adds error", func(t *testing.T) {
		def := &ServiceDefinition{
			Metadata: ServiceMetadata{
				Name:     "test",
				Version:  "1.0.0",
				Category: CategoryMedia,
			},
			Spec: ServiceSpec{
				Image:      ImageSpec{Repository: "test/image"},
				Container:  ContainerSpec{NameTemplate: "{{ .Name }}"},
				Networking: NetworkSpec{Mode: "host"},
			},
		}

		trust := TrustLevel{AllowHostNetwork: false}
		errors := v.ValidateWithTrustLevel(def, trust)

		// Should have trust level error (base only has warning)
		found := false
		for _, e := range errors {
			if e.Field == "spec.networking.mode" && e.Severity == "error" && e.Message == "host network not allowed by trust level" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected trust level error for host network, got: %v", errors)
		}
	})

	t.Run("capability not allowed", func(t *testing.T) {
		def := &ServiceDefinition{
			Metadata: ServiceMetadata{
				Name:     "test",
				Version:  "1.0.0",
				Category: CategoryMedia,
			},
			Spec: ServiceSpec{
				Image: ImageSpec{Repository: "test/image"},
				Container: ContainerSpec{
					NameTemplate: "{{ .Name }}",
					Capabilities: CapabilitiesSpec{Add: []string{"NET_ADMIN"}},
				},
			},
		}

		trust := TrustLevel{AllowCapabilities: []string{"SYS_TIME"}}
		errors := v.ValidateWithTrustLevel(def, trust)

		// Should have error for capability not allowed
		found := false
		for _, e := range errors {
			if e.Field == "spec.container.capabilities.add" && e.Severity == "error" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error for capability not allowed, got: %v", errors)
		}
	})

	t.Run("capability allowed with wildcard", func(t *testing.T) {
		def := &ServiceDefinition{
			Metadata: ServiceMetadata{
				Name:     "test",
				Version:  "1.0.0",
				Category: CategoryMedia,
			},
			Spec: ServiceSpec{
				Image: ImageSpec{Repository: "test/image"},
				Container: ContainerSpec{
					NameTemplate: "{{ .Name }}",
					Capabilities: CapabilitiesSpec{Add: []string{"NET_ADMIN"}},
				},
			},
		}

		trust := TrustLevel{AllowCapabilities: []string{"*"}}
		errors := v.ValidateWithTrustLevel(def, trust)

		// Should NOT have trust level error for capability (wildcard allows all)
		for _, e := range errors {
			if e.Field == "spec.container.capabilities.add" && e.Severity == "error" && e.Message != "" {
				// NET_ADMIN is not a dangerous cap, so no base warning either
				if e.Message == "capability NET_ADMIN not allowed by trust level" {
					t.Errorf("wildcard should allow all capabilities, got: %v", errors)
				}
			}
		}
	})

	t.Run("registry not allowed", func(t *testing.T) {
		def := &ServiceDefinition{
			Metadata: ServiceMetadata{
				Name:     "test",
				Version:  "1.0.0",
				Category: CategoryMedia,
			},
			Spec: ServiceSpec{
				Image:     ImageSpec{Repository: "test/image", Registry: "custom.io"},
				Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
			},
		}

		trust := TrustLevel{AllowedRegistries: []string{"docker.io"}}
		errors := v.ValidateWithTrustLevel(def, trust)

		// Should have error for registry not allowed
		found := false
		for _, e := range errors {
			if e.Field == "spec.image.registry" && e.Severity == "error" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error for registry not allowed, got: %v", errors)
		}
	})

	t.Run("registry allowed with wildcard", func(t *testing.T) {
		def := &ServiceDefinition{
			Metadata: ServiceMetadata{
				Name:     "test",
				Version:  "1.0.0",
				Category: CategoryMedia,
			},
			Spec: ServiceSpec{
				Image:     ImageSpec{Repository: "test/image", Registry: "custom.io"},
				Container: ContainerSpec{NameTemplate: "{{ .Name }}"},
			},
		}

		trust := TrustLevel{AllowedRegistries: []string{"*"}}
		errors := v.ValidateWithTrustLevel(def, trust)

		// Should NOT have trust level error for registry (wildcard allows all)
		// May still have base warning for unallowed registry
		for _, e := range errors {
			if e.Field == "spec.image.registry" && e.Severity == "error" {
				t.Errorf("wildcard should allow all registries, got error: %v", e)
			}
		}
	})
}

// TestHasErrors verifies HasErrors function
func TestHasErrors(t *testing.T) {
	tests := []struct {
		name   string
		errors []ValidationError
		want   bool
	}{
		{
			name:   "no errors",
			errors: []ValidationError{},
			want:   false,
		},
		{
			name:   "only warnings",
			errors: []ValidationError{{Severity: "warning"}},
			want:   false,
		},
		{
			name:   "has error",
			errors: []ValidationError{{Severity: "error"}},
			want:   true,
		},
		{
			name: "mixed",
			errors: []ValidationError{
				{Severity: "warning"},
				{Severity: "error"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasErrors(tt.errors)
			if got != tt.want {
				t.Errorf("HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFilterBySeverity verifies FilterBySeverity function
func TestFilterBySeverity(t *testing.T) {
	errors := []ValidationError{
		{Field: "a", Severity: "error"},
		{Field: "b", Severity: "warning"},
		{Field: "c", Severity: "error"},
		{Field: "d", Severity: "info"},
	}

	errorOnly := FilterBySeverity(errors, "error")
	if len(errorOnly) != 2 {
		t.Errorf("expected 2 errors, got %d", len(errorOnly))
	}

	warningOnly := FilterBySeverity(errors, "warning")
	if len(warningOnly) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warningOnly))
	}

	infoOnly := FilterBySeverity(errors, "info")
	if len(infoOnly) != 1 {
		t.Errorf("expected 1 info, got %d", len(infoOnly))
	}
}

// TestIsValidServiceName verifies service name validation
func TestIsValidServiceName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"a", true},
		{"test", true},
		{"test-service", true},
		{"my-cool-app-123", true},
		{"abc123", true},
		{"", false},
		{"Test", false},
		{"UPPER", false},
		{"test_underscore", false},
		{"-starts-with-dash", false},
		{"ends-with-dash-", false},
		{"123starts-with-number", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidServiceName(tt.name)
			if got != tt.valid {
				t.Errorf("isValidServiceName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

// TestIsValidSubdomain verifies subdomain validation
func TestIsValidSubdomain(t *testing.T) {
	tests := []struct {
		subdomain string
		valid     bool
	}{
		{"a", true},
		{"test", true},
		{"my-app", true},
		{"app123", true},
		{"", false},
		{"Test", false},
		{"-invalid", false},
		{"invalid-", false},
	}

	for _, tt := range tests {
		t.Run(tt.subdomain, func(t *testing.T) {
			got := isValidSubdomain(tt.subdomain)
			if got != tt.valid {
				t.Errorf("isValidSubdomain(%q) = %v, want %v", tt.subdomain, got, tt.valid)
			}
		})
	}
}

// TestIsValidCategory verifies category validation
func TestIsValidCategory(t *testing.T) {
	validCategories := []ServiceCategory{
		CategoryMedia,
		CategoryDownloads,
		CategoryManagement,
		CategoryUtility,
		CategoryNetworking,
		CategoryAuth,
	}

	for _, cat := range validCategories {
		if !isValidCategory(cat) {
			t.Errorf("expected category %s to be valid", cat)
		}
	}

	invalidCategories := []ServiceCategory{"invalid", "unknown", ""}
	for _, cat := range invalidCategories {
		if isValidCategory(cat) {
			t.Errorf("expected category %s to be invalid", cat)
		}
	}
}
