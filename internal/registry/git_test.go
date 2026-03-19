package registry

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsValidSSHKeyPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"normal path", "/home/user/.ssh/id_rsa", true},
		{"path with spaces", "/home/user/.ssh/my key", true},
		{"tilde path", "~/.ssh/id_rsa", false}, // tilde is not in allowed chars (should be expanded before calling)
		{"relative path", ".ssh/id_rsa", true},
		{"path with dash", "/home/user/.ssh/id-ed25519", true},
		{"path with underscore", "/home/user/.ssh/id_ed25519", true},
		{"empty path", "", false},
		{"backtick injection", "/home/user/`whoami`", false},
		{"dollar injection", "/home/user/$HOME/key", false},
		{"semicolon injection", "/home/user/key;rm -rf /", false},
		{"pipe injection", "/home/user/key|cat /etc/passwd", false},
		{"single quote injection", "/home/user/key'", false},
		{"double quote injection", `/home/user/key"`, false},
		{"newline injection", "/home/user/key\nwhoami", false},
		{"ampersand injection", "/home/user/key&whoami", false},
		{"parenthesis injection", "/home/user/key()", false},
		{"just alphanumeric", "id_rsa", true},
		{"windows-like path", "C/Users/test/.ssh/id_rsa", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSSHKeyPath(tt.path)
			if result != tt.expected {
				t.Errorf("isValidSSHKeyPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestNewGitSource(t *testing.T) {
	cache := NewCache(t.TempDir())

	src := Source{
		Name:     "test-git",
		Type:     "git",
		URL:      "https://github.com/test/repo.git",
		Branch:   "main",
		SSHKey:   "/home/user/.ssh/id_rsa",
		Path:     "services",
		Priority: 10,
		Enabled:  true,
		Verified: true,
	}

	gs := NewGitSource(src, cache)

	if gs.Name() != "test-git" {
		t.Errorf("Name() = %q, want %q", gs.Name(), "test-git")
	}
	if gs.Type() != "git" {
		t.Errorf("Type() = %q, want %q", gs.Type(), "git")
	}
	if gs.Priority() != 10 {
		t.Errorf("Priority() = %d, want %d", gs.Priority(), 10)
	}
	if !gs.IsEnabled() {
		t.Error("IsEnabled() should be true")
	}
	if gs.GetURL() != "https://github.com/test/repo.git" {
		t.Errorf("GetURL() = %q", gs.GetURL())
	}
	if gs.GetBranch() != "main" {
		t.Errorf("GetBranch() = %q", gs.GetBranch())
	}
	if !gs.IsVerified() {
		t.Error("IsVerified() should be true")
	}
	if gs.GetCommit() != "" {
		t.Errorf("GetCommit() should be empty initially, got %q", gs.GetCommit())
	}
}

func TestGitSourceGetServicesPath(t *testing.T) {
	cache := NewCache(t.TempDir())

	tests := []struct {
		name    string
		subPath string
		want    string // suffix that the path should end with
	}{
		{
			name:    "no subpath",
			subPath: "",
			want:    "test-src",
		},
		{
			name:    "with subpath",
			subPath: "services",
			want:    filepath.Join("test-src", "services"),
		},
		{
			name:    "nested subpath",
			subPath: "path/to/services",
			want:    filepath.Join("test-src", "path", "to", "services"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := NewGitSource(Source{
				Name:    "test-src",
				Type:    "git",
				Path:    tt.subPath,
				Enabled: true,
			}, cache)

			path := gs.getServicesPath()
			if !strings.HasSuffix(path, tt.want) {
				t.Errorf("getServicesPath() = %q, want suffix %q", path, tt.want)
			}
		})
	}
}

func TestGitSourceGitCommand(t *testing.T) {
	cache := NewCache(t.TempDir())

	t.Run("basic command without SSH key", func(t *testing.T) {
		gs := NewGitSource(Source{
			Name:    "test",
			Type:    "git",
			Enabled: true,
		}, cache)

		ctx := context.Background()
		cmd := gs.gitCommand(ctx, "/tmp/test", "status")

		if cmd.Dir != "/tmp/test" {
			t.Errorf("Dir = %q, want /tmp/test", cmd.Dir)
		}
		if cmd.Args[0] != "git" || cmd.Args[1] != "status" {
			t.Errorf("Args = %v, want [git status]", cmd.Args)
		}
		// Should not have custom env
		if cmd.Env != nil {
			t.Error("Env should be nil when no SSH key")
		}
	})

	t.Run("command with SSH key", func(t *testing.T) {
		gs := NewGitSource(Source{
			Name:    "test",
			Type:    "git",
			SSHKey:  "/home/user/.ssh/id_rsa",
			Enabled: true,
		}, cache)

		ctx := context.Background()
		cmd := gs.gitCommand(ctx, "", "clone", "url")

		// Should have GIT_SSH_COMMAND in env containing our SSH key path.
		// Check the LAST GIT_SSH_COMMAND entry since os.Environ() may already have one,
		// and our append comes last (which is what exec uses).
		var lastSSHCmd string
		for _, env := range cmd.Env {
			if strings.HasPrefix(env, "GIT_SSH_COMMAND=") {
				lastSSHCmd = env
			}
		}
		if lastSSHCmd == "" {
			t.Error("expected GIT_SSH_COMMAND in environment")
		} else {
			if !strings.Contains(lastSSHCmd, "/home/user/.ssh/id_rsa") {
				t.Errorf("GIT_SSH_COMMAND should contain SSH key path, got %q", lastSSHCmd)
			}
			if !strings.Contains(lastSSHCmd, "StrictHostKeyChecking=accept-new") {
				t.Errorf("GIT_SSH_COMMAND should use StrictHostKeyChecking=accept-new, got %q", lastSSHCmd)
			}
		}
	})

	t.Run("command with invalid SSH key path", func(t *testing.T) {
		gs := NewGitSource(Source{
			Name:    "test",
			Type:    "git",
			SSHKey:  "/home/user/key`whoami`",
			Enabled: true,
		}, cache)

		ctx := context.Background()
		cmd := gs.gitCommand(ctx, "", "clone", "url")

		// Should NOT have GIT_SSH_COMMAND since path is invalid
		for _, env := range cmd.Env {
			if strings.HasPrefix(env, "GIT_SSH_COMMAND=") {
				t.Error("should not set GIT_SSH_COMMAND with invalid SSH key path")
			}
		}
	})

	t.Run("empty dir uses no dir", func(t *testing.T) {
		gs := NewGitSource(Source{
			Name:    "test",
			Type:    "git",
			Enabled: true,
		}, cache)

		ctx := context.Background()
		cmd := gs.gitCommand(ctx, "", "status")

		if cmd.Dir != "" {
			t.Errorf("Dir should be empty, got %q", cmd.Dir)
		}
	})
}

func TestGitSourceIsCloned(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	gs := NewGitSource(Source{
		Name:    "test-repo",
		Type:    "git",
		Enabled: true,
	}, cache)

	// Initially not cloned
	if gs.isCloned() {
		t.Error("should not be cloned initially")
	}

	// Create a fake .git directory
	repoPath := cache.GetRepoPath("test-repo")
	gitDir := filepath.Join(repoPath, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatal(err)
	}

	if !gs.isCloned() {
		t.Error("should be cloned after creating .git dir")
	}
}

// initTestGitRepo creates a minimal git repo with service definitions for testing.
func initTestGitRepo(t *testing.T, dir string, services map[string]string) {
	t.Helper()

	// Initialize git repo
	cmd := exec.Command("git", "init", dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s: %v", output, err)
	}

	// Configure git user for commit
	for _, args := range [][]string{
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	} {
		cmd = exec.Command("git", args...)
		cmd.Dir = dir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git config failed: %s: %v", output, err)
		}
	}

	// Create service files
	for path, content := range services {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Commit
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %s: %v", output, err)
	}

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %s: %v", output, err)
	}
}

func TestGitSourceLoadServiceFromClonedRepo(t *testing.T) {
	// Create a "remote" repo
	remoteDir := t.TempDir()
	svcYAML := `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: test-svc
  version: 1.0.0
  category: utility
  description: Test service
spec:
  image:
    repository: test/image
    tag: latest
  container:
    name_template: "sdbx-test-svc"
routing:
  enabled: false
conditions:
  always: true
`
	initTestGitRepo(t, remoteDir, map[string]string{
		"core/test-svc/service.yaml": svcYAML,
	})

	// Create git source pointing to local "remote"
	cacheDir := t.TempDir()
	cache := NewCache(cacheDir)

	gs := NewGitSource(Source{
		Name:    "test-local-git",
		Type:    "git",
		URL:     remoteDir,
		Branch:  "master",
		Enabled: true,
	}, cache)

	ctx := context.Background()

	// Load the service (triggers clone)
	def, err := gs.LoadService(ctx, "test-svc")
	if err != nil {
		t.Fatalf("LoadService() error: %v", err)
	}

	if def.Metadata.Name != "test-svc" {
		t.Errorf("expected service name 'test-svc', got %q", def.Metadata.Name)
	}
	if def.Metadata.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %q", def.Metadata.Version)
	}

	// Commit should now be set
	if gs.GetCommit() == "" {
		t.Error("commit should be set after clone")
	}
}

func TestGitSourceListServicesFromClonedRepo(t *testing.T) {
	remoteDir := t.TempDir()
	svcA := `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: svc-a
  version: 1.0.0
  category: utility
  description: Service A
spec:
  image:
    repository: test/a
    tag: latest
  container:
    name_template: "sdbx-svc-a"
routing:
  enabled: false
conditions:
  always: true
`
	svcB := `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: svc-b
  version: 1.0.0
  category: media
  description: Service B
spec:
  image:
    repository: test/b
    tag: latest
  container:
    name_template: "sdbx-svc-b"
routing:
  enabled: false
conditions:
  requireAddon: true
`
	initTestGitRepo(t, remoteDir, map[string]string{
		"core/svc-a/service.yaml":   svcA,
		"addons/svc-b/service.yaml": svcB,
	})

	cacheDir := t.TempDir()
	cache := NewCache(cacheDir)

	gs := NewGitSource(Source{
		Name:    "test-list",
		Type:    "git",
		URL:     remoteDir,
		Branch:  "master",
		Enabled: true,
	}, cache)

	ctx := context.Background()
	services, err := gs.ListServices(ctx)
	if err != nil {
		t.Fatalf("ListServices() error: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("expected 2 services, got %d: %v", len(services), services)
	}
}

func TestGitSourceGetServicePath(t *testing.T) {
	remoteDir := t.TempDir()
	svcYAML := `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: my-svc
  version: 1.0.0
  category: utility
  description: Test
spec:
  image:
    repository: test/img
    tag: latest
  container:
    name_template: "sdbx-my-svc"
routing:
  enabled: false
conditions:
  always: true
`
	initTestGitRepo(t, remoteDir, map[string]string{
		"addons/my-svc/service.yaml": svcYAML,
	})

	cacheDir := t.TempDir()
	cache := NewCache(cacheDir)

	gs := NewGitSource(Source{
		Name:    "test-path",
		Type:    "git",
		URL:     remoteDir,
		Branch:  "master",
		Enabled: true,
	}, cache)

	// Clone first
	ctx := context.Background()
	_, _ = gs.ListServices(ctx)

	path := gs.GetServicePath("my-svc")
	if !strings.Contains(path, "addons/my-svc/service.yaml") {
		t.Errorf("GetServicePath() = %q, want to contain addons/my-svc/service.yaml", path)
	}
}

func TestGitSourceHasService(t *testing.T) {
	remoteDir := t.TempDir()
	svcYAML := `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: found-svc
  version: 1.0.0
  category: utility
  description: Found
spec:
  image:
    repository: test/img
    tag: latest
  container:
    name_template: "sdbx-found-svc"
routing:
  enabled: false
conditions:
  always: true
`
	initTestGitRepo(t, remoteDir, map[string]string{
		"core/found-svc/service.yaml": svcYAML,
	})

	cacheDir := t.TempDir()
	cache := NewCache(cacheDir)

	gs := NewGitSource(Source{
		Name:    "test-has",
		Type:    "git",
		URL:     remoteDir,
		Branch:  "master",
		Enabled: true,
	}, cache)

	ctx := context.Background()

	if !gs.HasService(ctx, "found-svc") {
		t.Error("HasService() should return true for existing service")
	}

	if gs.HasService(ctx, "missing-svc") {
		t.Error("HasService() should return false for missing service")
	}
}

func TestGitSourceUpdateCommitHash(t *testing.T) {
	remoteDir := t.TempDir()
	initTestGitRepo(t, remoteDir, map[string]string{
		"README.md": "test",
	})

	cacheDir := t.TempDir()
	cache := NewCache(cacheDir)

	gs := NewGitSource(Source{
		Name:    "test-commit",
		Type:    "git",
		URL:     remoteDir,
		Branch:  "master",
		Enabled: true,
	}, cache)

	// Clone first
	ctx := context.Background()
	if err := gs.clone(ctx); err != nil {
		t.Fatalf("clone() error: %v", err)
	}

	// Commit should be set
	commit := gs.GetCommit()
	if commit == "" {
		t.Error("commit should not be empty after clone")
	}
	if len(commit) != 40 {
		t.Errorf("commit hash should be 40 chars, got %d: %q", len(commit), commit)
	}
}

func TestGitSourceLoadServiceNotFound(t *testing.T) {
	remoteDir := t.TempDir()
	initTestGitRepo(t, remoteDir, map[string]string{
		"core/existing/service.yaml": `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: existing
  version: 1.0.0
  category: utility
  description: Existing
spec:
  image:
    repository: test/img
    tag: latest
  container:
    name_template: "sdbx-existing"
routing:
  enabled: false
conditions:
  always: true
`,
	})

	cacheDir := t.TempDir()
	cache := NewCache(cacheDir)

	gs := NewGitSource(Source{
		Name:    "test-notfound",
		Type:    "git",
		URL:     remoteDir,
		Branch:  "master",
		Enabled: true,
	}, cache)

	ctx := context.Background()
	_, err := gs.LoadService(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent service")
	}
}

func TestGitSourceWithSubPath(t *testing.T) {
	remoteDir := t.TempDir()
	svcYAML := `apiVersion: sdbx.one/v1
kind: Service
metadata:
  name: sub-svc
  version: 1.0.0
  category: utility
  description: Subpath service
spec:
  image:
    repository: test/sub
    tag: latest
  container:
    name_template: "sdbx-sub-svc"
routing:
  enabled: false
conditions:
  always: true
`
	initTestGitRepo(t, remoteDir, map[string]string{
		"nested/services/core/sub-svc/service.yaml": svcYAML,
	})

	cacheDir := t.TempDir()
	cache := NewCache(cacheDir)

	gs := NewGitSource(Source{
		Name:    "test-subpath",
		Type:    "git",
		URL:     remoteDir,
		Branch:  "master",
		Path:    "nested/services",
		Enabled: true,
	}, cache)

	ctx := context.Background()
	def, err := gs.LoadService(ctx, "sub-svc")
	if err != nil {
		t.Fatalf("LoadService() with subpath error: %v", err)
	}

	if def.Metadata.Name != "sub-svc" {
		t.Errorf("expected name 'sub-svc', got %q", def.Metadata.Name)
	}
}
