package registry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/maiko/sdbx/internal/config"
)

// Registry manages service definitions from multiple sources
type Registry struct {
	sources   []SourceProvider
	cache     *Cache
	validator *Validator
	resolver  *Resolver
	mu        sync.RWMutex
}

// SourceProvider is the interface for service definition sources
type SourceProvider interface {
	// Name returns the source name
	Name() string

	// Type returns the source type (local, git)
	Type() string

	// Priority returns the source priority (higher = checked first)
	Priority() int

	// IsEnabled returns whether the source is enabled
	IsEnabled() bool

	// Load loads all service definitions from the source
	Load(ctx context.Context) ([]*ServiceDefinition, error)

	// LoadService loads a specific service definition
	LoadService(ctx context.Context, name string) (*ServiceDefinition, error)

	// ListServices returns names of all available services
	ListServices(ctx context.Context) ([]string, error)

	// GetServicePath returns the path to a service definition
	GetServicePath(name string) string

	// Update updates the source (e.g., git pull)
	Update(ctx context.Context) error

	// GetCommit returns the current commit hash (for git sources)
	GetCommit() string
}

// New creates a new Registry with the given configuration
func New(cfg *SourceConfig) (*Registry, error) {
	r := &Registry{
		sources:   make([]SourceProvider, 0),
		validator: NewValidator(),
	}

	// Initialize cache
	cacheDir := cfg.Cache.Directory
	if cacheDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		cacheDir = filepath.Join(home, ".cache", "sdbx", "sources")
	}
	r.cache = NewCache(cacheDir)

	// Initialize sources
	for _, src := range cfg.Sources {
		if !src.Enabled {
			continue
		}

		provider, err := r.createSourceProvider(src)
		if err != nil {
			return nil, fmt.Errorf("failed to create source %s: %w", src.Name, err)
		}
		r.sources = append(r.sources, provider)
	}

	// Always add embedded source as a fallback (lowest priority)
	embeddedSource := NewEmbeddedSource()
	r.sources = append(r.sources, embeddedSource)

	// Sort sources by priority (highest first)
	sort.Slice(r.sources, func(i, j int) bool {
		return r.sources[i].Priority() > r.sources[j].Priority()
	})

	// Initialize resolver
	r.resolver = NewResolver(r)

	return r, nil
}

// NewWithDefaults creates a Registry with default configuration
func NewWithDefaults() (*Registry, error) {
	cfg := DefaultSourceConfig()
	return New(cfg)
}

// DefaultSourceConfig returns the default source configuration
func DefaultSourceConfig() *SourceConfig {
	home, _ := os.UserHomeDir()

	return &SourceConfig{
		APIVersion: APIVersion,
		Kind:       KindSourceConfig,
		Metadata: SourceConfigMetadata{
			Version: 1,
		},
		Sources: []Source{
			{
				Name:     "local",
				Type:     "local",
				Path:     filepath.Join(home, ".config", "sdbx", "services"),
				Priority: 100,
				Enabled:  true,
			},
			{
				Name:     "official",
				Type:     "git",
				URL:      "https://github.com/maiko/SDBX-Services.git",
				Branch:   "main",
				Path:     "services",
				Priority: 0,
				Enabled:  true,
				Verified: true,
			},
		},
		Cache: CacheConfig{
			Directory: filepath.Join(home, ".cache", "sdbx", "sources"),
			TTL:       "24h",
		},
		Security: SecurityConfig{
			AllowUnverified: true,
		},
	}
}

// createSourceProvider creates a source provider based on source config
func (r *Registry) createSourceProvider(src Source) (SourceProvider, error) {
	switch src.Type {
	case "local":
		return NewLocalSource(src), nil
	case "git":
		return NewGitSource(src, r.cache), nil
	case "embedded":
		return NewEmbeddedSource(), nil
	default:
		return nil, fmt.Errorf("unknown source type: %s", src.Type)
	}
}

// Sources returns all configured source providers
func (r *Registry) Sources() []SourceProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sources
}

// AddSource adds a new source to the registry
func (r *Registry) AddSource(src Source) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicate
	for _, existing := range r.sources {
		if existing.Name() == src.Name {
			return fmt.Errorf("source %s already exists", src.Name)
		}
	}

	provider, err := r.createSourceProvider(src)
	if err != nil {
		return err
	}

	r.sources = append(r.sources, provider)

	// Re-sort by priority
	sort.Slice(r.sources, func(i, j int) bool {
		return r.sources[i].Priority() > r.sources[j].Priority()
	})

	return nil
}

// RemoveSource removes a source from the registry
func (r *Registry) RemoveSource(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, src := range r.sources {
		if src.Name() == name {
			r.sources = append(r.sources[:i], r.sources[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("source %s not found", name)
}

// GetSource returns a source by name
func (r *Registry) GetSource(name string) (SourceProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, src := range r.sources {
		if src.Name() == name {
			return src, nil
		}
	}

	return nil, fmt.Errorf("source %s not found", name)
}

// Update updates all sources
func (r *Registry) Update(ctx context.Context) error {
	r.mu.RLock()
	sources := r.sources
	r.mu.RUnlock()

	var errs []error
	for _, src := range sources {
		if err := src.Update(ctx); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", src.Name(), err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("update errors: %v", errs)
	}
	return nil
}

// Resolve resolves all services based on the project configuration
func (r *Registry) Resolve(ctx context.Context, cfg *config.Config) (*ResolutionGraph, error) {
	return r.resolver.Resolve(ctx, cfg)
}

// GetService returns a service definition by name (searches all sources by priority)
func (r *Registry) GetService(ctx context.Context, name string) (*ServiceDefinition, string, error) {
	r.mu.RLock()
	sources := r.sources
	r.mu.RUnlock()

	for _, src := range sources {
		if !src.IsEnabled() {
			continue
		}

		def, err := src.LoadService(ctx, name)
		if err == nil && def != nil {
			return def, src.Name(), nil
		}
	}

	return nil, "", fmt.Errorf("service %s not found in any source", name)
}

// ListServices returns all available services across all sources
func (r *Registry) ListServices(ctx context.Context) ([]ServiceInfo, error) {
	r.mu.RLock()
	sources := r.sources
	r.mu.RUnlock()

	seen := make(map[string]bool)
	var services []ServiceInfo

	for _, src := range sources {
		if !src.IsEnabled() {
			continue
		}

		names, err := src.ListServices(ctx)
		if err != nil {
			continue
		}

		for _, name := range names {
			if seen[name] {
				continue
			}
			seen[name] = true

			def, err := src.LoadService(ctx, name)
			if err != nil {
				continue
			}

			services = append(services, ServiceInfo{
				Name:        def.Metadata.Name,
				Description: def.Metadata.Description,
				Category:    def.Metadata.Category,
				Version:     def.Metadata.Version,
				Source:      src.Name(),
				IsAddon:     def.Conditions.RequireAddon,
				HasWebUI:    def.Routing.Enabled,
			})
		}
	}

	// Sort by name
	sort.Slice(services, func(i, j int) bool {
		return services[i].Name < services[j].Name
	})

	return services, nil
}

// SearchServices searches for services matching a query
func (r *Registry) SearchServices(ctx context.Context, query string, category ServiceCategory) ([]ServiceInfo, error) {
	all, err := r.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	var results []ServiceInfo
	for _, svc := range all {
		if category != "" && svc.Category != category {
			continue
		}

		if matchesQuery(svc, query) {
			results = append(results, svc)
		}
	}

	return results, nil
}

// Validate validates a service definition
func (r *Registry) Validate(def *ServiceDefinition) []ValidationError {
	return r.validator.Validate(def)
}

// ServiceInfo provides summary information about a service
type ServiceInfo struct {
	Name        string
	Description string
	Category    ServiceCategory
	Version     string
	Source      string
	IsAddon     bool
	HasWebUI    bool
}

// matchesQuery checks if a service matches a search query
func matchesQuery(svc ServiceInfo, query string) bool {
	if query == "" {
		return true
	}

	// Simple case-insensitive substring match
	query = toLower(query)
	if contains(toLower(svc.Name), query) {
		return true
	}
	if contains(toLower(svc.Description), query) {
		return true
	}
	if contains(toLower(string(svc.Category)), query) {
		return true
	}

	return false
}

// LockFileDiff represents a difference in lock file comparison
type LockFileDiff struct {
	Type        string // "added", "removed", "changed"
	Description string
}

// GenerateLockFile generates a lock file from current configuration
func (r *Registry) GenerateLockFile(ctx context.Context, cfg *config.Config) (*LockFile, error) {
	// Resolve all services
	graph, err := r.Resolve(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve services: %w", err)
	}

	// Build lock file
	lock := &LockFile{
		APIVersion: APIVersion,
		Kind:       KindLockFile,
		Metadata: LockFileMetadata{
			Version:     1,
			GeneratedAt: time.Now().UTC(),
		},
		Sources:      make(map[string]LockedSource),
		Services:     make(map[string]LockedService),
		InstallOrder: graph.Order,
	}

	// Lock sources
	for _, src := range r.Sources() {
		if gitSrc, ok := src.(*GitSource); ok {
			lock.Sources[src.Name()] = LockedSource{
				URL:       gitSrc.GetURL(),
				Commit:    gitSrc.GetCommit(),
				Branch:    gitSrc.GetBranch(),
				FetchedAt: gitSrc.GetLastUpdated(),
			}
		}
	}

	// Lock services
	for name, resolved := range graph.Services {
		if !resolved.Enabled {
			continue
		}

		def := resolved.FinalDefinition
		lock.Services[name] = LockedService{
			Source:            resolved.Source,
			DefinitionVersion: def.Metadata.Version,
			Image: LockedImage{
				Repository: def.Spec.Image.Repository,
				Tag:        def.Spec.Image.Tag,
			},
			ResolvedFrom: resolved.SourcePath,
			Enabled:      resolved.Enabled,
		}
	}

	return lock, nil
}

// DiffLockFiles compares two lock files and returns differences
func (r *Registry) DiffLockFiles(existing, current *LockFile) []LockFileDiff {
	var diffs []LockFileDiff

	// Compare sources
	for name, lockedSrc := range existing.Sources {
		if currentSrc, exists := current.Sources[name]; exists {
			if lockedSrc.Commit != currentSrc.Commit {
				diffs = append(diffs, LockFileDiff{
					Type:        "changed",
					Description: fmt.Sprintf("Source %s: commit changed from %s to %s", name, truncateCommit(lockedSrc.Commit), truncateCommit(currentSrc.Commit)),
				})
			}
		} else {
			diffs = append(diffs, LockFileDiff{
				Type:        "removed",
				Description: fmt.Sprintf("Source %s: removed", name),
			})
		}
	}

	for name := range current.Sources {
		if _, exists := existing.Sources[name]; !exists {
			diffs = append(diffs, LockFileDiff{
				Type:        "added",
				Description: fmt.Sprintf("Source %s: added", name),
			})
		}
	}

	// Compare services
	for name, lockedSvc := range existing.Services {
		if currentSvc, exists := current.Services[name]; exists {
			if lockedSvc.DefinitionVersion != currentSvc.DefinitionVersion {
				diffs = append(diffs, LockFileDiff{
					Type:        "changed",
					Description: fmt.Sprintf("Service %s: version changed from %s to %s", name, lockedSvc.DefinitionVersion, currentSvc.DefinitionVersion),
				})
			}
			if lockedSvc.Image.Tag != currentSvc.Image.Tag {
				diffs = append(diffs, LockFileDiff{
					Type:        "changed",
					Description: fmt.Sprintf("Service %s: image tag changed from %s to %s", name, lockedSvc.Image.Tag, currentSvc.Image.Tag),
				})
			}
		} else {
			diffs = append(diffs, LockFileDiff{
				Type:        "removed",
				Description: fmt.Sprintf("Service %s: removed", name),
			})
		}
	}

	for name := range current.Services {
		if _, exists := existing.Services[name]; !exists {
			diffs = append(diffs, LockFileDiff{
				Type:        "added",
				Description: fmt.Sprintf("Service %s: added", name),
			})
		}
	}

	return diffs
}

// UpdateLockFile updates services in the lock file
func (r *Registry) UpdateLockFile(ctx context.Context, cfg *config.Config, existing *LockFile, servicesToUpdate []string) (*LockFile, error) {
	// Generate new lock file
	current, err := r.GenerateLockFile(ctx, cfg)
	if err != nil {
		return nil, err
	}

	// If no specific services, return the fully regenerated lock file
	if len(servicesToUpdate) == 0 {
		return current, nil
	}

	// Only update specified services
	updated := &LockFile{
		APIVersion:   existing.APIVersion,
		Kind:         existing.Kind,
		Metadata:     current.Metadata, // Update metadata
		Sources:      existing.Sources,
		Services:     make(map[string]LockedService),
		InstallOrder: current.InstallOrder,
	}

	// Copy existing services, update only specified ones
	for name, svc := range existing.Services {
		if containsString(servicesToUpdate, name) {
			if newSvc, exists := current.Services[name]; exists {
				updated.Services[name] = newSvc
			} else {
				// Service no longer exists, remove it
				continue
			}
		} else {
			updated.Services[name] = svc
		}
	}

	// Add any new services that weren't in existing
	for name, svc := range current.Services {
		if _, exists := updated.Services[name]; !exists {
			updated.Services[name] = svc
		}
	}

	return updated, nil
}

// truncateCommit truncates a commit hash for display
func truncateCommit(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

// containsString checks if slice contains string
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// toLower converts string to lowercase
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr) >= 0)
}

// findSubstring finds substr in s
func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
