package registry

import (
	"fmt"
	"regexp"
	"strings"
)

// Validator validates service definitions
type Validator struct {
	allowedRegistries map[string]bool
	dangerousCaps     map[string]bool
}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	return &Validator{
		allowedRegistries: map[string]bool{
			"docker.io":   true,
			"ghcr.io":     true,
			"lscr.io":     true,
			"quay.io":     true,
			"gcr.io":      true,
			"registry.k8s.io": true,
		},
		dangerousCaps: map[string]bool{
			"SYS_ADMIN":   true,
			"SYS_PTRACE":  true,
			"SYS_MODULE":  true,
			"SYS_RAWIO":   true,
			"SYS_TIME":    true,
			"DAC_READ_SEARCH": true,
		},
	}
}

// Validate validates a service definition and returns all errors
func (v *Validator) Validate(def *ServiceDefinition) []ValidationError {
	var errors []ValidationError

	// Validate metadata
	errors = append(errors, v.validateMetadata(def)...)

	// Validate spec
	errors = append(errors, v.validateSpec(def)...)

	// Validate routing
	errors = append(errors, v.validateRouting(def)...)

	// Validate security
	errors = append(errors, v.validateSecurity(def)...)

	return errors
}

// validateMetadata validates service metadata
func (v *Validator) validateMetadata(def *ServiceDefinition) []ValidationError {
	var errors []ValidationError

	if def.Metadata.Name == "" {
		errors = append(errors, ValidationError{
			Field:    "metadata.name",
			Message:  "name is required",
			Severity: "error",
		})
	} else if !isValidServiceName(def.Metadata.Name) {
		errors = append(errors, ValidationError{
			Field:    "metadata.name",
			Message:  "name must be lowercase alphanumeric with hyphens",
			Severity: "error",
		})
	}

	if def.Metadata.Version == "" {
		errors = append(errors, ValidationError{
			Field:    "metadata.version",
			Message:  "version is required",
			Severity: "error",
		})
	}

	if def.Metadata.Category == "" {
		errors = append(errors, ValidationError{
			Field:    "metadata.category",
			Message:  "category is required",
			Severity: "error",
		})
	} else if !isValidCategory(def.Metadata.Category) {
		errors = append(errors, ValidationError{
			Field:    "metadata.category",
			Message:  fmt.Sprintf("invalid category: %s", def.Metadata.Category),
			Severity: "error",
		})
	}

	if def.Metadata.Description == "" {
		errors = append(errors, ValidationError{
			Field:    "metadata.description",
			Message:  "description is recommended",
			Severity: "warning",
		})
	}

	return errors
}

// validateSpec validates the service spec
func (v *Validator) validateSpec(def *ServiceDefinition) []ValidationError {
	var errors []ValidationError

	// Validate image
	if def.Spec.Image.Repository == "" {
		errors = append(errors, ValidationError{
			Field:    "spec.image.repository",
			Message:  "image repository is required",
			Severity: "error",
		})
	}

	// Validate container name template
	if def.Spec.Container.NameTemplate == "" {
		errors = append(errors, ValidationError{
			Field:    "spec.container.name_template",
			Message:  "container name template is required",
			Severity: "error",
		})
	} else if !strings.Contains(def.Spec.Container.NameTemplate, "{{") {
		errors = append(errors, ValidationError{
			Field:    "spec.container.name_template",
			Message:  "container name template should use Go template syntax",
			Severity: "warning",
		})
	}

	// Validate volumes
	for i, vol := range def.Spec.Volumes {
		if vol.HostPath == "" {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.volumes[%d].hostPath", i),
				Message:  "hostPath is required",
				Severity: "error",
			})
		}
		if vol.ContainerPath == "" {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.volumes[%d].containerPath", i),
				Message:  "containerPath is required",
				Severity: "error",
			})
		}
	}

	// Validate environment variables
	for i, env := range def.Spec.Environment.Static {
		if env.Name == "" {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.environment.static[%d].name", i),
				Message:  "environment variable name is required",
				Severity: "error",
			})
		}
		if env.Value == "" && env.ValueFrom == nil {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.environment.static[%d]", i),
				Message:  "environment variable must have value or valueFrom",
				Severity: "error",
			})
		}
	}

	// Validate conditional environment variables
	for i, env := range def.Spec.Environment.Conditional {
		if env.Name == "" {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.environment.conditional[%d].name", i),
				Message:  "environment variable name is required",
				Severity: "error",
			})
		}
		if env.When == "" {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.environment.conditional[%d].when", i),
				Message:  "conditional environment variable must have 'when' condition",
				Severity: "error",
			})
		}
	}

	// Validate health check
	if def.Spec.HealthCheck != nil {
		if len(def.Spec.HealthCheck.Test) == 0 {
			errors = append(errors, ValidationError{
				Field:    "spec.healthcheck.test",
				Message:  "health check test command is required",
				Severity: "error",
			})
		}
	}

	// Validate dependencies
	for i, dep := range def.Spec.Dependencies.Conditional {
		if dep.Name == "" {
			errors = append(errors, ValidationError{
				Field:    fmt.Sprintf("spec.dependencies.conditional[%d].name", i),
				Message:  "dependency name is required",
				Severity: "error",
			})
		}
	}

	return errors
}

// validateRouting validates routing configuration
func (v *Validator) validateRouting(def *ServiceDefinition) []ValidationError {
	var errors []ValidationError

	if !def.Routing.Enabled {
		return errors
	}

	if def.Routing.Port <= 0 || def.Routing.Port > 65535 {
		errors = append(errors, ValidationError{
			Field:    "routing.port",
			Message:  "port must be between 1 and 65535",
			Severity: "error",
		})
	}

	if def.Routing.Subdomain != "" && !isValidSubdomain(def.Routing.Subdomain) {
		errors = append(errors, ValidationError{
			Field:    "routing.subdomain",
			Message:  "subdomain must be lowercase alphanumeric with hyphens",
			Severity: "error",
		})
	}

	if def.Routing.Path != "" && !strings.HasPrefix(def.Routing.Path, "/") {
		errors = append(errors, ValidationError{
			Field:    "routing.path",
			Message:  "path must start with /",
			Severity: "error",
		})
	}

	validStrategies := map[string]bool{
		"stripPrefix": true,
		"urlBase":     true,
		"none":        true,
		"":            true,
	}
	if !validStrategies[def.Routing.PathRouting.Strategy] {
		errors = append(errors, ValidationError{
			Field:    "routing.pathRouting.strategy",
			Message:  fmt.Sprintf("invalid strategy: %s", def.Routing.PathRouting.Strategy),
			Severity: "error",
		})
	}

	return errors
}

// validateSecurity validates security-related settings
func (v *Validator) validateSecurity(def *ServiceDefinition) []ValidationError {
	var errors []ValidationError

	// Check for privileged mode
	if def.Spec.Container.Privileged {
		errors = append(errors, ValidationError{
			Field:    "spec.container.privileged",
			Message:  "privileged mode is a security risk",
			Severity: "error",
		})
	}

	// Check for dangerous capabilities
	for _, cap := range def.Spec.Container.Capabilities.Add {
		if v.dangerousCaps[cap] {
			errors = append(errors, ValidationError{
				Field:    "spec.container.capabilities.add",
				Message:  fmt.Sprintf("dangerous capability %s requires explicit approval", cap),
				Severity: "warning",
			})
		}
	}

	// Check for host network mode
	if def.Spec.Networking.Mode == "host" {
		errors = append(errors, ValidationError{
			Field:    "spec.networking.mode",
			Message:  "host network mode bypasses network isolation",
			Severity: "warning",
		})
	}

	// Check image source
	registry := def.Spec.Image.Registry
	if registry == "" {
		registry = "docker.io"
	}
	if !v.allowedRegistries[registry] {
		errors = append(errors, ValidationError{
			Field:    "spec.image.registry",
			Message:  fmt.Sprintf("registry %s is not in allowed list", registry),
			Severity: "warning",
		})
	}

	// Check for potentially dangerous devices
	for _, device := range def.Spec.Container.Devices {
		if strings.Contains(device, "/dev/mem") || strings.Contains(device, "/dev/kmem") {
			errors = append(errors, ValidationError{
				Field:    "spec.container.devices",
				Message:  fmt.Sprintf("dangerous device mapping: %s", device),
				Severity: "error",
			})
		}
	}

	return errors
}

// ValidateWithTrustLevel validates against a specific trust level
func (v *Validator) ValidateWithTrustLevel(def *ServiceDefinition, trust TrustLevel) []ValidationError {
	errors := v.Validate(def)

	// Check privileged
	if def.Spec.Container.Privileged && !trust.AllowPrivileged {
		errors = append(errors, ValidationError{
			Field:    "spec.container.privileged",
			Message:  "privileged mode not allowed by trust level",
			Severity: "error",
		})
	}

	// Check host network
	if def.Spec.Networking.Mode == "host" && !trust.AllowHostNetwork {
		errors = append(errors, ValidationError{
			Field:    "spec.networking.mode",
			Message:  "host network not allowed by trust level",
			Severity: "error",
		})
	}

	// Check capabilities
	allowedCaps := make(map[string]bool)
	for _, cap := range trust.AllowCapabilities {
		if cap == "*" {
			allowedCaps["*"] = true
			break
		}
		allowedCaps[cap] = true
	}

	if !allowedCaps["*"] {
		for _, cap := range def.Spec.Container.Capabilities.Add {
			if !allowedCaps[cap] {
				errors = append(errors, ValidationError{
					Field:    "spec.container.capabilities.add",
					Message:  fmt.Sprintf("capability %s not allowed by trust level", cap),
					Severity: "error",
				})
			}
		}
	}

	// Check registries
	allowedRegs := make(map[string]bool)
	for _, reg := range trust.AllowedRegistries {
		if reg == "*" {
			allowedRegs["*"] = true
			break
		}
		allowedRegs[reg] = true
	}

	if !allowedRegs["*"] {
		registry := def.Spec.Image.Registry
		if registry == "" {
			registry = "docker.io"
		}
		if !allowedRegs[registry] {
			errors = append(errors, ValidationError{
				Field:    "spec.image.registry",
				Message:  fmt.Sprintf("registry %s not allowed by trust level", registry),
				Severity: "error",
			})
		}
	}

	return errors
}

// HasErrors returns true if there are any error-severity validation errors
func HasErrors(errors []ValidationError) bool {
	for _, e := range errors {
		if e.Severity == "error" {
			return true
		}
	}
	return false
}

// FilterByServerity filters validation errors by severity
func FilterBySeverity(errors []ValidationError, severity string) []ValidationError {
	var filtered []ValidationError
	for _, e := range errors {
		if e.Severity == severity {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// isValidServiceName checks if a name is a valid service name
func isValidServiceName(name string) bool {
	// Must be lowercase, alphanumeric, with hyphens
	matched, _ := regexp.MatchString(`^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$`, name)
	return matched
}

// isValidSubdomain checks if a subdomain is valid
func isValidSubdomain(subdomain string) bool {
	matched, _ := regexp.MatchString(`^[a-z][a-z0-9-]*[a-z0-9]$|^[a-z]$`, subdomain)
	return matched
}

// isValidCategory checks if a category is valid
func isValidCategory(category ServiceCategory) bool {
	valid := map[ServiceCategory]bool{
		CategoryMedia:      true,
		CategoryDownloads:  true,
		CategoryManagement: true,
		CategoryUtility:    true,
		CategoryNetworking: true,
		CategoryAuth:       true,
	}
	return valid[category]
}
