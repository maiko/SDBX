package registry

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitSource implements SourceProvider for Git repository sources
type GitSource struct {
	BaseSource
	url      string
	branch   string
	sshKey   string
	subPath  string
	cache    *Cache
	commit   string
	verified bool
}

// NewGitSource creates a new Git source
func NewGitSource(src Source, cache *Cache) *GitSource {
	return &GitSource{
		BaseSource: BaseSource{
			name:     src.Name,
			srcType:  "git",
			priority: src.Priority,
			enabled:  src.Enabled,
			loader:   NewLoader(),
		},
		url:      src.URL,
		branch:   src.Branch,
		sshKey:   src.SSHKey,
		subPath:  src.Path,
		cache:    cache,
		verified: src.Verified,
	}
}

// Load loads all service definitions from the Git source
func (s *GitSource) Load(ctx context.Context) ([]*ServiceDefinition, error) {
	// Ensure repo is cloned/updated
	if err := s.ensureCloned(ctx); err != nil {
		return nil, err
	}

	servicesPath := s.getServicesPath()
	return s.loader.LoadServicesFromDir(servicesPath)
}

// LoadService loads a specific service definition
func (s *GitSource) LoadService(ctx context.Context, name string) (*ServiceDefinition, error) {
	if err := s.ensureCloned(ctx); err != nil {
		return nil, err
	}

	servicesPath := s.getServicesPath()

	// Try direct path
	path := filepath.Join(servicesPath, name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return s.loader.LoadServiceDefinition(path)
	}

	// Try core/ subdirectory
	path = filepath.Join(servicesPath, "core", name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return s.loader.LoadServiceDefinition(path)
	}

	// Try addons/ subdirectory
	path = filepath.Join(servicesPath, "addons", name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return s.loader.LoadServiceDefinition(path)
	}

	return nil, fmt.Errorf("service %s not found in source %s", name, s.name)
}

// ListServices returns names of all available services
func (s *GitSource) ListServices(ctx context.Context) ([]string, error) {
	if err := s.ensureCloned(ctx); err != nil {
		return nil, err
	}

	servicesPath := s.getServicesPath()
	return s.loader.DiscoverServices(servicesPath)
}

// GetServicePath returns the path to a service definition
func (s *GitSource) GetServicePath(name string) string {
	servicesPath := s.getServicesPath()

	// Check direct path
	path := filepath.Join(servicesPath, name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Check core/
	path = filepath.Join(servicesPath, "core", name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Check addons/
	path = filepath.Join(servicesPath, "addons", name, "service.yaml")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return filepath.Join(servicesPath, name, "service.yaml")
}

// Update updates the Git repository
func (s *GitSource) Update(ctx context.Context) error {
	repoPath := s.cache.GetRepoPath(s.name)

	if !s.isCloned() {
		return s.clone(ctx)
	}

	// Git pull
	cmd := s.gitCommand(ctx, repoPath, "pull", "origin", s.branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull failed: %s: %w", string(output), err)
	}

	// Update commit hash
	return s.updateCommitHash(ctx)
}

// GetCommit returns the current commit hash
func (s *GitSource) GetCommit() string {
	return s.commit
}

// IsVerified returns whether this source is verified/official
func (s *GitSource) IsVerified() bool {
	return s.verified
}

// GetURL returns the Git repository URL
func (s *GitSource) GetURL() string {
	return s.url
}

// GetBranch returns the branch name
func (s *GitSource) GetBranch() string {
	return s.branch
}

// ensureCloned ensures the repository is cloned and up to date
func (s *GitSource) ensureCloned(ctx context.Context) error {
	if s.isCloned() {
		// Check if we need to update
		if s.cache.NeedsUpdate(s.name) {
			return s.Update(ctx)
		}
		return s.updateCommitHash(ctx)
	}

	return s.clone(ctx)
}

// isCloned checks if the repository is already cloned
func (s *GitSource) isCloned() bool {
	repoPath := s.cache.GetRepoPath(s.name)
	gitPath := filepath.Join(repoPath, ".git")
	_, err := os.Stat(gitPath)
	return err == nil
}

// clone clones the Git repository
func (s *GitSource) clone(ctx context.Context) error {
	repoPath := s.cache.GetRepoPath(s.name)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Remove existing directory if it exists
	os.RemoveAll(repoPath)

	// Clone
	args := []string{"clone", "--branch", s.branch, "--single-branch", "--depth", "1", s.url, repoPath}
	cmd := s.gitCommand(ctx, "", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %s: %w", string(output), err)
	}

	// Update cache timestamp
	s.cache.MarkUpdated(s.name)

	return s.updateCommitHash(ctx)
}

// updateCommitHash gets and stores the current commit hash
func (s *GitSource) updateCommitHash(ctx context.Context) error {
	repoPath := s.cache.GetRepoPath(s.name)

	cmd := s.gitCommand(ctx, repoPath, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get commit hash: %w", err)
	}

	s.commit = strings.TrimSpace(string(output))
	return nil
}

// gitCommand creates a git command with optional SSH key
func (s *GitSource) gitCommand(ctx context.Context, dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	// Set up SSH key if provided
	if s.sshKey != "" {
		// Expand ~ in path
		sshKey := s.sshKey
		if len(sshKey) > 0 && sshKey[0] == '~' {
			home, _ := os.UserHomeDir()
			sshKey = filepath.Join(home, sshKey[1:])
		}

		env := os.Environ()
		env = append(env, fmt.Sprintf("GIT_SSH_COMMAND=ssh -i %s -o StrictHostKeyChecking=no", sshKey))
		cmd.Env = env
	}

	return cmd
}

// getServicesPath returns the path to the services directory
func (s *GitSource) getServicesPath() string {
	repoPath := s.cache.GetRepoPath(s.name)
	if s.subPath != "" {
		return filepath.Join(repoPath, s.subPath)
	}
	return repoPath
}

// GetRepoMetadata loads the repository metadata
func (s *GitSource) GetRepoMetadata(ctx context.Context) (*SourceRepository, error) {
	if err := s.ensureCloned(ctx); err != nil {
		return nil, err
	}

	repoPath := s.cache.GetRepoPath(s.name)
	metaPath := filepath.Join(repoPath, "sources.yaml")

	if _, err := os.Stat(metaPath); err != nil {
		return nil, fmt.Errorf("repository metadata not found")
	}

	return s.loader.LoadSourceRepository(metaPath)
}

// Fetch fetches updates without merging
func (s *GitSource) Fetch(ctx context.Context) error {
	if !s.isCloned() {
		return s.clone(ctx)
	}

	repoPath := s.cache.GetRepoPath(s.name)
	cmd := s.gitCommand(ctx, repoPath, "fetch", "origin", s.branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch failed: %s: %w", string(output), err)
	}

	return nil
}

// GetLastUpdated returns when the source was last updated
func (s *GitSource) GetLastUpdated() time.Time {
	return s.cache.GetLastUpdated(s.name)
}

// HasService checks if a service exists in this source
func (s *GitSource) HasService(ctx context.Context, name string) bool {
	if err := s.ensureCloned(ctx); err != nil {
		return false
	}

	servicesPath := s.getServicesPath()
	paths := []string{
		filepath.Join(servicesPath, name, "service.yaml"),
		filepath.Join(servicesPath, "core", name, "service.yaml"),
		filepath.Join(servicesPath, "addons", name, "service.yaml"),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}
