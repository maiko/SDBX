package registry

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/maiko/sdbx/internal/config"
	"gopkg.in/yaml.v3"
)

// LockManager handles lock file operations
type LockManager struct {
	registry   *Registry
	loader     *Loader
	cliVersion string
}

// NewLockManager creates a new LockManager
func NewLockManager(registry *Registry, cliVersion string) *LockManager {
	return &LockManager{
		registry:   registry,
		loader:     NewLoader(),
		cliVersion: cliVersion,
	}
}

// GenerateLockFile generates a new lock file from the current configuration
func (m *LockManager) GenerateLockFile(ctx context.Context, cfg *config.Config, outputPath string) (*LockFile, error) {
	// Resolve all services
	graph, err := m.registry.Resolve(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve services: %w", err)
	}

	// Calculate config hash
	configHash, err := m.calculateConfigHash(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate config hash: %w", err)
	}

	// Build lock file
	lock := &LockFile{
		APIVersion: APIVersion,
		Kind:       KindLockFile,
		Metadata: LockFileMetadata{
			Version:     1,
			GeneratedAt: time.Now().UTC(),
			CLIVersion:  m.cliVersion,
			ConfigHash:  configHash,
		},
		Sources:        make(map[string]LockedSource),
		Services:       make(map[string]LockedService),
		InstallOrder:   graph.Order,
		GeneratedFiles: make(map[string]string),
	}

	// Lock sources
	for _, src := range m.registry.Sources() {
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

	// Save lock file
	if outputPath != "" {
		if err := m.loader.SaveLockFile(outputPath, lock); err != nil {
			return nil, fmt.Errorf("failed to save lock file: %w", err)
		}
	}

	return lock, nil
}

// LoadLockFile loads an existing lock file
func (m *LockManager) LoadLockFile(path string) (*LockFile, error) {
	return m.loader.LoadLockFile(path)
}

// Verify verifies that the lock file matches the current state
func (m *LockManager) Verify(ctx context.Context, cfg *config.Config, lock *LockFile) ([]LockVerificationResult, error) {
	var results []LockVerificationResult

	// Verify config hash
	currentHash, err := m.calculateConfigHash(cfg)
	if err != nil {
		return nil, err
	}

	if currentHash != lock.Metadata.ConfigHash {
		results = append(results, LockVerificationResult{
			Type:     "config",
			Status:   "changed",
			Message:  "Configuration has changed since lock file was generated",
			Expected: lock.Metadata.ConfigHash,
			Actual:   currentHash,
		})
	}

	// Verify sources
	for sourceName, locked := range lock.Sources {
		src, err := m.registry.GetSource(sourceName)
		if err != nil {
			results = append(results, LockVerificationResult{
				Type:    "source",
				Name:    sourceName,
				Status:  "missing",
				Message: "Source not found",
			})
			continue
		}

		if gitSrc, ok := src.(*GitSource); ok {
			currentCommit := gitSrc.GetCommit()
			if currentCommit != locked.Commit {
				results = append(results, LockVerificationResult{
					Type:     "source",
					Name:     sourceName,
					Status:   "changed",
					Message:  "Source commit has changed",
					Expected: locked.Commit,
					Actual:   currentCommit,
				})
			}
		}
	}

	// Verify services
	for serviceName, locked := range lock.Services {
		if !locked.Enabled {
			continue
		}

		def, source, err := m.registry.GetService(ctx, serviceName)
		if err != nil {
			results = append(results, LockVerificationResult{
				Type:    "service",
				Name:    serviceName,
				Status:  "missing",
				Message: "Service not found",
			})
			continue
		}

		// Check source
		if source != locked.Source {
			results = append(results, LockVerificationResult{
				Type:     "service",
				Name:     serviceName,
				Status:   "changed",
				Message:  "Service source changed",
				Expected: locked.Source,
				Actual:   source,
			})
		}

		// Check version
		if def.Metadata.Version != locked.DefinitionVersion {
			results = append(results, LockVerificationResult{
				Type:     "service",
				Name:     serviceName,
				Status:   "changed",
				Message:  "Service definition version changed",
				Expected: locked.DefinitionVersion,
				Actual:   def.Metadata.Version,
			})
		}

		// Check image
		if def.Spec.Image.Repository != locked.Image.Repository {
			results = append(results, LockVerificationResult{
				Type:     "service",
				Name:     serviceName,
				Status:   "changed",
				Message:  "Image repository changed",
				Expected: locked.Image.Repository,
				Actual:   def.Spec.Image.Repository,
			})
		}

		if def.Spec.Image.Tag != locked.Image.Tag {
			results = append(results, LockVerificationResult{
				Type:     "service",
				Name:     serviceName,
				Status:   "changed",
				Message:  "Image tag changed",
				Expected: locked.Image.Tag,
				Actual:   def.Spec.Image.Tag,
			})
		}
	}

	return results, nil
}

// Diff compares lock file with current state and returns differences
func (m *LockManager) Diff(ctx context.Context, cfg *config.Config, lock *LockFile) (*LockDiff, error) {
	// Generate what the lock file would be now
	current, err := m.GenerateLockFile(ctx, cfg, "")
	if err != nil {
		return nil, err
	}

	diff := &LockDiff{
		Sources:  make(map[string]DiffEntry),
		Services: make(map[string]DiffEntry),
	}

	// Compare sources
	for name, locked := range lock.Sources {
		if current, exists := current.Sources[name]; exists {
			if locked.Commit != current.Commit {
				diff.Sources[name] = DiffEntry{
					Type:     "modified",
					Old:      locked.Commit,
					New:      current.Commit,
				}
			}
		} else {
			diff.Sources[name] = DiffEntry{
				Type: "removed",
				Old:  locked.Commit,
			}
		}
	}

	for name, current := range current.Sources {
		if _, exists := lock.Sources[name]; !exists {
			diff.Sources[name] = DiffEntry{
				Type: "added",
				New:  current.Commit,
			}
		}
	}

	// Compare services
	for name, locked := range lock.Services {
		if current, exists := current.Services[name]; exists {
			if locked.DefinitionVersion != current.DefinitionVersion {
				diff.Services[name] = DiffEntry{
					Type: "modified",
					Old:  locked.DefinitionVersion,
					New:  current.DefinitionVersion,
				}
			}
		} else {
			diff.Services[name] = DiffEntry{
				Type: "removed",
				Old:  locked.DefinitionVersion,
			}
		}
	}

	for name, current := range current.Services {
		if _, exists := lock.Services[name]; !exists {
			diff.Services[name] = DiffEntry{
				Type: "added",
				New:  current.DefinitionVersion,
			}
		}
	}

	return diff, nil
}

// Update updates the lock file to latest versions
func (m *LockManager) Update(ctx context.Context, cfg *config.Config, lock *LockFile, outputPath string) (*LockFile, error) {
	// Update all sources first
	if err := m.registry.Update(ctx); err != nil {
		return nil, fmt.Errorf("failed to update sources: %w", err)
	}

	// Generate new lock file
	return m.GenerateLockFile(ctx, cfg, outputPath)
}

// calculateConfigHash calculates a hash of the configuration
func (m *LockManager) calculateConfigHash(cfg *config.Config) (string, error) {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", hash[:16]), nil
}

// LockVerificationResult represents a lock file verification result
type LockVerificationResult struct {
	Type     string // "config", "source", "service"
	Name     string
	Status   string // "ok", "changed", "missing"
	Message  string
	Expected string
	Actual   string
}

// LockDiff represents differences between lock file and current state
type LockDiff struct {
	Sources  map[string]DiffEntry
	Services map[string]DiffEntry
}

// DiffEntry represents a single diff entry
type DiffEntry struct {
	Type string // "added", "removed", "modified"
	Old  string
	New  string
}

// HasChanges returns true if there are any differences
func (d *LockDiff) HasChanges() bool {
	return len(d.Sources) > 0 || len(d.Services) > 0
}

// IsEmpty returns true if there are no changes
func (d *LockDiff) IsEmpty() bool {
	return !d.HasChanges()
}

// GetLockFilePath returns the default lock file path for a project
func GetLockFilePath(projectDir string) string {
	return filepath.Join(projectDir, ".sdbx.lock")
}

// LockFileExists checks if a lock file exists
func LockFileExists(projectDir string) bool {
	path := GetLockFilePath(projectDir)
	_, err := os.Stat(path)
	return err == nil
}
