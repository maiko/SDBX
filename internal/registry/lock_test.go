package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLockDiffHasChanges tests LockDiff change detection
func TestLockDiffHasChanges(t *testing.T) {
	tests := []struct {
		name       string
		diff       LockDiff
		hasChanges bool
	}{
		{
			name:       "empty diff",
			diff:       LockDiff{},
			hasChanges: false,
		},
		{
			name: "only source changes",
			diff: LockDiff{
				Sources: map[string]DiffEntry{
					"source1": {Type: "modified"},
				},
			},
			hasChanges: true,
		},
		{
			name: "only service changes",
			diff: LockDiff{
				Services: map[string]DiffEntry{
					"service1": {Type: "added"},
				},
			},
			hasChanges: true,
		},
		{
			name: "both changes",
			diff: LockDiff{
				Sources: map[string]DiffEntry{
					"source1": {Type: "modified"},
				},
				Services: map[string]DiffEntry{
					"service1": {Type: "removed"},
				},
			},
			hasChanges: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.diff.HasChanges() != tt.hasChanges {
				t.Errorf("HasChanges() = %v, want %v", tt.diff.HasChanges(), tt.hasChanges)
			}
		})
	}
}

// TestLockDiffIsEmpty tests LockDiff empty detection
func TestLockDiffIsEmpty(t *testing.T) {
	tests := []struct {
		name    string
		diff    LockDiff
		isEmpty bool
	}{
		{
			name:    "empty diff",
			diff:    LockDiff{},
			isEmpty: true,
		},
		{
			name: "with sources",
			diff: LockDiff{
				Sources: map[string]DiffEntry{
					"source1": {Type: "added"},
				},
			},
			isEmpty: false,
		},
		{
			name: "with services",
			diff: LockDiff{
				Services: map[string]DiffEntry{
					"service1": {Type: "removed"},
				},
			},
			isEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.diff.IsEmpty() != tt.isEmpty {
				t.Errorf("IsEmpty() = %v, want %v", tt.diff.IsEmpty(), tt.isEmpty)
			}
		})
	}
}

// TestDiffEntry tests DiffEntry struct
func TestDiffEntry(t *testing.T) {
	entry := DiffEntry{
		Type: "modified",
		Old:  "1.0.0",
		New:  "2.0.0",
	}

	if entry.Type != "modified" {
		t.Errorf("Type = %q, want 'modified'", entry.Type)
	}

	if entry.Old != "1.0.0" {
		t.Errorf("Old = %q, want '1.0.0'", entry.Old)
	}

	if entry.New != "2.0.0" {
		t.Errorf("New = %q, want '2.0.0'", entry.New)
	}
}

// TestGetLockFilePath tests lock file path generation
func TestGetLockFilePath(t *testing.T) {
	path := GetLockFilePath("/home/user/project")
	expected := filepath.Join("/home/user/project", ".sdbx.lock")

	if path != expected {
		t.Errorf("GetLockFilePath = %q, want %q", path, expected)
	}
}

// TestLockFileExists tests lock file existence check
func TestLockFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not exist initially
	if LockFileExists(tmpDir) {
		t.Error("lock file should not exist initially")
	}

	// Create lock file
	lockPath := GetLockFilePath(tmpDir)
	if err := os.WriteFile(lockPath, []byte("test"), 0o644); err != nil {
		t.Fatalf("failed to create lock file: %v", err)
	}

	// Should exist now
	if !LockFileExists(tmpDir) {
		t.Error("lock file should exist after creation")
	}
}

// TestLockVerificationResult tests verification result struct
func TestLockVerificationResult(t *testing.T) {
	result := LockVerificationResult{
		Type:     "service",
		Name:     "nginx",
		Status:   "changed",
		Message:  "Image tag changed",
		Expected: "1.0.0",
		Actual:   "2.0.0",
	}

	if result.Type != "service" {
		t.Errorf("Type = %q, want 'service'", result.Type)
	}

	if result.Name != "nginx" {
		t.Errorf("Name = %q, want 'nginx'", result.Name)
	}

	if result.Status != "changed" {
		t.Errorf("Status = %q, want 'changed'", result.Status)
	}

	if result.Expected != "1.0.0" {
		t.Errorf("Expected = %q, want '1.0.0'", result.Expected)
	}
}

// TestNewLockManager tests LockManager creation
func TestNewLockManager(t *testing.T) {
	// LockManager needs a Registry, but we can test with nil for basic creation
	manager := NewLockManager(nil, "1.0.0")

	if manager == nil {
		t.Fatal("NewLockManager returned nil")
	}

	if manager.cliVersion != "1.0.0" {
		t.Errorf("cliVersion = %q, want '1.0.0'", manager.cliVersion)
	}

	if manager.loader == nil {
		t.Error("loader should not be nil")
	}
}

// TestLockManagerLoadLockFile tests loading lock files
func TestLockManagerLoadLockFile(t *testing.T) {
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
	lockPath := filepath.Join(tmpDir, ".sdbx.lock")
	if err := os.WriteFile(lockPath, []byte(lockYAML), 0o644); err != nil {
		t.Fatalf("failed to write lock file: %v", err)
	}

	manager := NewLockManager(nil, "1.0.0")
	lock, err := manager.LoadLockFile(lockPath)
	if err != nil {
		t.Fatalf("LoadLockFile failed: %v", err)
	}

	if lock.Metadata.CLIVersion != "1.0.0" {
		t.Errorf("CLIVersion = %q, want '1.0.0'", lock.Metadata.CLIVersion)
	}

	if lock.Metadata.ConfigHash != "sha256:abc123" {
		t.Errorf("ConfigHash = %q, want 'sha256:abc123'", lock.Metadata.ConfigHash)
	}

	if len(lock.Sources) != 1 {
		t.Errorf("sources count = %d, want 1", len(lock.Sources))
	}

	if lock.Services["nginx"].DefinitionVersion != "1.0.0" {
		t.Errorf("nginx version = %q, want '1.0.0'", lock.Services["nginx"].DefinitionVersion)
	}
}

// TestLockManagerLoadLockFileNotFound tests error for missing lock file
func TestLockManagerLoadLockFileNotFound(t *testing.T) {
	manager := NewLockManager(nil, "1.0.0")
	_, err := manager.LoadLockFile("/nonexistent/path/.sdbx.lock")

	if err == nil {
		t.Error("LoadLockFile should return error for non-existent file")
	}
}

// TestLockFileStruct tests LockFile struct
func TestLockFileStruct(t *testing.T) {
	now := time.Now()

	lock := LockFile{
		APIVersion: "sdbx.io/v1",
		Kind:       "LockFile",
		Metadata: LockFileMetadata{
			Version:     1,
			GeneratedAt: now,
			CLIVersion:  "1.0.0",
			ConfigHash:  "sha256:abc123",
		},
		Sources: map[string]LockedSource{
			"official": {
				URL:       "https://github.com/example/services",
				Commit:    "abc123",
				Branch:    "main",
				FetchedAt: now,
			},
		},
		Services: map[string]LockedService{
			"nginx": {
				Source:            "official",
				DefinitionVersion: "1.0.0",
				Image: LockedImage{
					Repository: "nginx",
					Tag:        "latest",
				},
				ResolvedFrom: "/path/to/service.yaml",
				Enabled:      true,
			},
		},
		InstallOrder: []string{"nginx"},
		GeneratedFiles: map[string]string{
			"compose.yaml": "sha256:xyz789",
		},
	}

	if lock.APIVersion != "sdbx.io/v1" {
		t.Errorf("APIVersion = %q, want 'sdbx.io/v1'", lock.APIVersion)
	}

	if lock.Metadata.Version != 1 {
		t.Errorf("Version = %d, want 1", lock.Metadata.Version)
	}

	if lock.Sources["official"].Commit != "abc123" {
		t.Errorf("Commit = %q, want 'abc123'", lock.Sources["official"].Commit)
	}

	if !lock.Services["nginx"].Enabled {
		t.Error("nginx should be enabled")
	}

	if len(lock.InstallOrder) != 1 {
		t.Errorf("InstallOrder length = %d, want 1", len(lock.InstallOrder))
	}
}

// TestLockedSourceStruct tests LockedSource struct
func TestLockedSourceStruct(t *testing.T) {
	now := time.Now()

	source := LockedSource{
		URL:       "https://github.com/example/services",
		Commit:    "abc123def456",
		Branch:    "main",
		FetchedAt: now,
	}

	if source.URL != "https://github.com/example/services" {
		t.Errorf("URL = %q, want git URL", source.URL)
	}

	if source.Commit != "abc123def456" {
		t.Errorf("Commit = %q, want 'abc123def456'", source.Commit)
	}

	if source.Branch != "main" {
		t.Errorf("Branch = %q, want 'main'", source.Branch)
	}
}

// TestLockedServiceStruct tests LockedService struct
func TestLockedServiceStruct(t *testing.T) {
	service := LockedService{
		Source:            "official",
		DefinitionVersion: "2.0.0",
		Image: LockedImage{
			Repository: "linuxserver/sonarr",
			Tag:        "4.0.0",
		},
		ResolvedFrom: "/path/to/sonarr/service.yaml",
		Enabled:      true,
	}

	if service.Source != "official" {
		t.Errorf("Source = %q, want 'official'", service.Source)
	}

	if service.Image.Repository != "linuxserver/sonarr" {
		t.Errorf("Repository = %q, want 'linuxserver/sonarr'", service.Image.Repository)
	}

	if service.Image.Tag != "4.0.0" {
		t.Errorf("Tag = %q, want '4.0.0'", service.Image.Tag)
	}
}

// TestLockedImageStruct tests LockedImage struct
func TestLockedImageStruct(t *testing.T) {
	image := LockedImage{
		Repository: "nginx",
		Tag:        "alpine",
	}

	if image.Repository != "nginx" {
		t.Errorf("Repository = %q, want 'nginx'", image.Repository)
	}

	if image.Tag != "alpine" {
		t.Errorf("Tag = %q, want 'alpine'", image.Tag)
	}
}
