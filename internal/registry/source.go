package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// BaseSource provides common functionality for source implementations
type BaseSource struct {
	name     string
	srcType  string
	priority int
	enabled  bool
	path     string
	loader   *Loader
}

// Name returns the source name
func (s *BaseSource) Name() string {
	return s.name
}

// Type returns the source type
func (s *BaseSource) Type() string {
	return s.srcType
}

// Priority returns the source priority
func (s *BaseSource) Priority() int {
	return s.priority
}

// IsEnabled returns whether the source is enabled
func (s *BaseSource) IsEnabled() bool {
	return s.enabled
}

// LocalSource implements SourceProvider for local filesystem sources
type LocalSource struct {
	BaseSource
}

// NewLocalSource creates a new local filesystem source
func NewLocalSource(src Source) *LocalSource {
	path := src.Path
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".config", "sdbx", "services")
	}

	// Expand ~ in path
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}

	return &LocalSource{
		BaseSource: BaseSource{
			name:     src.Name,
			srcType:  "local",
			priority: src.Priority,
			enabled:  src.Enabled,
			path:     path,
			loader:   NewLoader(),
		},
	}
}

// Load loads all service definitions from the local source
func (s *LocalSource) Load(ctx context.Context) ([]*ServiceDefinition, error) {
	if !s.exists() {
		return nil, nil
	}

	return s.loader.LoadServicesFromDir(s.path)
}

// LoadService loads a specific service definition
func (s *LocalSource) LoadService(ctx context.Context, name string) (*ServiceDefinition, error) {
	if !s.exists() {
		return nil, fmt.Errorf("source directory does not exist: %s", s.path)
	}

	// Try direct path first
	path := filepath.Join(s.path, name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return s.loader.LoadServiceDefinition(path)
	}

	// Try core/ subdirectory
	path = filepath.Join(s.path, "core", name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return s.loader.LoadServiceDefinition(path)
	}

	// Try addons/ subdirectory
	path = filepath.Join(s.path, "addons", name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return s.loader.LoadServiceDefinition(path)
	}

	return nil, fmt.Errorf("service %s not found in source %s", name, s.name)
}

// ListServices returns names of all available services
func (s *LocalSource) ListServices(ctx context.Context) ([]string, error) {
	if !s.exists() {
		return nil, nil
	}

	return s.loader.DiscoverServices(s.path)
}

// GetServicePath returns the path to a service definition
func (s *LocalSource) GetServicePath(name string) string {
	// Check direct path
	path := filepath.Join(s.path, name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Check core/
	path = filepath.Join(s.path, "core", name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Check addons/
	path = filepath.Join(s.path, "addons", name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return filepath.Join(s.path, name, "service.yaml")
}

// Update is a no-op for local sources
func (s *LocalSource) Update(ctx context.Context) error {
	return nil
}

// GetCommit returns empty string for local sources
func (s *LocalSource) GetCommit() string {
	return ""
}

// exists checks if the source directory exists
func (s *LocalSource) exists() bool {
	_, err := os.Stat(s.path)
	return err == nil
}

// GetPath returns the base path of the source
func (s *LocalSource) GetPath() string {
	return s.path
}

// HasService checks if a service exists in this source
func (s *LocalSource) HasService(name string) bool {
	paths := []string{
		filepath.Join(s.path, name, "service.yaml"),
		filepath.Join(s.path, "core", name, "service.yaml"),
		filepath.Join(s.path, "addons", name, "service.yaml"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

// CreateServiceDir creates a directory for a new service
func (s *LocalSource) CreateServiceDir(name string, isAddon bool) (string, error) {
	var path string
	if isAddon {
		path = filepath.Join(s.path, "addons", name)
	} else {
		path = filepath.Join(s.path, "core", name)
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", fmt.Errorf("failed to create service directory: %w", err)
	}

	return path, nil
}

// SaveService saves a service definition to this source
func (s *LocalSource) SaveService(def *ServiceDefinition) error {
	isAddon := def.Conditions.RequireAddon

	dir, err := s.CreateServiceDir(def.Metadata.Name, isAddon)
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "service.yaml")
	return s.loader.SaveServiceDefinition(path, def)
}

// DeleteService deletes a service from this source
func (s *LocalSource) DeleteService(name string) error {
	paths := []string{
		filepath.Join(s.path, name),
		filepath.Join(s.path, "core", name),
		filepath.Join(s.path, "addons", name),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return os.RemoveAll(path)
		}
	}

	return fmt.Errorf("service %s not found", name)
}
