package registry

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Loader handles loading and parsing of YAML service definitions
type Loader struct{}

// NewLoader creates a new Loader
func NewLoader() *Loader {
	return &Loader{}
}

// LoadServiceDefinition loads a service definition from a file
func (l *Loader) LoadServiceDefinition(path string) (*ServiceDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return l.ParseServiceDefinition(data)
}

// ParseServiceDefinition parses a service definition from YAML data
func (l *Loader) ParseServiceDefinition(data []byte) (*ServiceDefinition, error) {
	var def ServiceDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate API version and kind
	if def.APIVersion != APIVersion {
		return nil, fmt.Errorf("unsupported API version: %s (expected %s)", def.APIVersion, APIVersion)
	}
	if def.Kind != KindService {
		return nil, fmt.Errorf("unexpected kind: %s (expected %s)", def.Kind, KindService)
	}

	// Apply defaults
	l.applyDefaults(&def)

	return &def, nil
}

// LoadServiceOverride loads a service override from a file
func (l *Loader) LoadServiceOverride(path string) (*ServiceOverride, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return l.ParseServiceOverride(data)
}

// ParseServiceOverride parses a service override from YAML data
func (l *Loader) ParseServiceOverride(data []byte) (*ServiceOverride, error) {
	var override ServiceOverride
	if err := yaml.Unmarshal(data, &override); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if override.APIVersion != APIVersion {
		return nil, fmt.Errorf("unsupported API version: %s", override.APIVersion)
	}
	if override.Kind != KindServiceOverride {
		return nil, fmt.Errorf("unexpected kind: %s (expected %s)", override.Kind, KindServiceOverride)
	}

	return &override, nil
}

// LoadSourceConfig loads a source configuration from a file
func (l *Loader) LoadSourceConfig(path string) (*SourceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return l.ParseSourceConfig(data)
}

// ParseSourceConfig parses a source configuration from YAML data
func (l *Loader) ParseSourceConfig(data []byte) (*SourceConfig, error) {
	var cfg SourceConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Set defaults for sources
	for i := range cfg.Sources {
		if cfg.Sources[i].Branch == "" && cfg.Sources[i].Type == "git" {
			cfg.Sources[i].Branch = "main"
		}
	}

	return &cfg, nil
}

// LoadLockFile loads a lock file
func (l *Loader) LoadLockFile(path string) (*LockFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return l.ParseLockFile(data)
}

// ParseLockFile parses a lock file from YAML data
func (l *Loader) ParseLockFile(data []byte) (*LockFile, error) {
	var lock LockFile
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if lock.APIVersion != APIVersion {
		return nil, fmt.Errorf("unsupported API version: %s", lock.APIVersion)
	}
	if lock.Kind != KindLockFile {
		return nil, fmt.Errorf("unexpected kind: %s (expected %s)", lock.Kind, KindLockFile)
	}

	return &lock, nil
}

// LoadSourceRepository loads source repository metadata
func (l *Loader) LoadSourceRepository(path string) (*SourceRepository, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	var repo SourceRepository
	if err := yaml.Unmarshal(data, &repo); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &repo, nil
}

// SaveServiceDefinition saves a service definition to a file
func (l *Loader) SaveServiceDefinition(path string, def *ServiceDefinition) error {
	return l.saveYAML(path, def)
}

// SaveSourceConfig saves a source configuration to a file
func (l *Loader) SaveSourceConfig(path string, cfg *SourceConfig) error {
	return l.saveYAML(path, cfg)
}

// SaveLockFile saves a lock file
func (l *Loader) SaveLockFile(path string, lock *LockFile) error {
	return l.saveYAML(path, lock)
}

// saveYAML saves data as YAML to a file
func (l *Loader) saveYAML(path string, data interface{}) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer f.Close()

	encoder := yaml.NewEncoder(f)
	encoder.SetIndent(2)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return nil
}

// applyDefaults applies default values to a service definition
func (l *Loader) applyDefaults(def *ServiceDefinition) {
	// Default container settings
	if def.Spec.Container.Restart == "" {
		def.Spec.Container.Restart = "unless-stopped"
	}
	if def.Spec.Container.NameTemplate == "" {
		def.Spec.Container.NameTemplate = "sdbx-{{ .Name }}"
	}

	// Default image registry
	if def.Spec.Image.Registry == "" {
		def.Spec.Image.Registry = "docker.io"
	}
	if def.Spec.Image.Tag == "" {
		def.Spec.Image.Tag = "latest"
	}

	// Default network mode
	if def.Spec.Networking.Mode == "" && def.Spec.Networking.ModeTemplate == "" {
		def.Spec.Networking.Mode = "bridge"
	}

	// Default routing path based on name
	if def.Routing.Enabled {
		if def.Routing.Subdomain == "" {
			def.Routing.Subdomain = def.Metadata.Name
		}
		if def.Routing.Path == "" {
			def.Routing.Path = "/" + def.Metadata.Name
		}
		if def.Routing.PathRouting.Strategy == "" {
			def.Routing.PathRouting.Strategy = "stripPrefix"
		}
	}

	// Default integrations
	if def.Integrations.Watchtower == nil {
		def.Integrations.Watchtower = &WatchtowerIntegration{Enabled: true}
	}
}

// DiscoverServices finds all service definitions in a directory
func (l *Loader) DiscoverServices(root string) ([]string, error) {
	var services []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if d.IsDir() && len(d.Name()) > 0 && d.Name()[0] == '.' {
			return filepath.SkipDir
		}

		// Look for service.yaml files
		if !d.IsDir() && d.Name() == "service.yaml" {
			// Get service name from parent directory
			serviceName := filepath.Base(filepath.Dir(path))
			services = append(services, serviceName)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	return services, nil
}

// LoadServicesFromDir loads all service definitions from a directory
func (l *Loader) LoadServicesFromDir(root string) ([]*ServiceDefinition, error) {
	services, err := l.DiscoverServices(root)
	if err != nil {
		return nil, err
	}

	var defs []*ServiceDefinition
	for _, name := range services {
		path := filepath.Join(root, name, "service.yaml")
		// Also check in core/ and addons/ subdirectories
		if _, err := os.Stat(path); os.IsNotExist(err) {
			path = filepath.Join(root, "core", name, "service.yaml")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				path = filepath.Join(root, "addons", name, "service.yaml")
			}
		}

		def, err := l.LoadServiceDefinition(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load service %s: %w", name, err)
		}
		defs = append(defs, def)
	}

	return defs, nil
}

// MergeOverride merges an override into a base service definition
func (l *Loader) MergeOverride(base *ServiceDefinition, override *ServiceOverride) *ServiceDefinition {
	// Create a deep copy of base
	merged := l.deepCopyServiceDefinition(base)

	if override.Spec != nil {
		// Merge image override
		if override.Spec.Image != nil {
			if override.Spec.Image.Repository != "" {
				merged.Spec.Image.Repository = override.Spec.Image.Repository
			}
			if override.Spec.Image.Tag != "" {
				merged.Spec.Image.Tag = override.Spec.Image.Tag
			}
			if override.Spec.Image.Registry != "" {
				merged.Spec.Image.Registry = override.Spec.Image.Registry
			}
		}

		// Merge environment additions
		if override.Spec.Environment != nil && len(override.Spec.Environment.Additional) > 0 {
			merged.Spec.Environment.Static = append(
				merged.Spec.Environment.Static,
				override.Spec.Environment.Additional...,
			)
		}

		// Merge volume additions
		if override.Spec.Volumes != nil && len(override.Spec.Volumes.Additional) > 0 {
			merged.Spec.Volumes = append(
				merged.Spec.Volumes,
				override.Spec.Volumes.Additional...,
			)
		}
	}

	// Merge routing override
	if override.Routing != nil {
		if override.Routing.Subdomain != nil {
			merged.Routing.Subdomain = *override.Routing.Subdomain
		}
		if override.Routing.Path != nil {
			merged.Routing.Path = *override.Routing.Path
		}
	}

	return merged
}

// deepCopyServiceDefinition creates a deep copy of a service definition
func (l *Loader) deepCopyServiceDefinition(def *ServiceDefinition) *ServiceDefinition {
	// Use YAML marshal/unmarshal for deep copy
	data, err := yaml.Marshal(def)
	if err != nil {
		// Fallback: return original if marshal fails
		return def
	}
	var copyDef ServiceDefinition
	if err := yaml.Unmarshal(data, &copyDef); err != nil {
		// Fallback: return original if unmarshal fails
		return def
	}
	return &copyDef
}

// WriteYAML writes data as YAML to a writer
func WriteYAML(w io.Writer, data interface{}) error {
	encoder := yaml.NewEncoder(w)
	encoder.SetIndent(2)
	return encoder.Encode(data)
}
