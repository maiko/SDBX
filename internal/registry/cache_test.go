package registry

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewCache tests Cache creation
func TestNewCache(t *testing.T) {
	tmpDir := t.TempDir()

	cache := NewCache(tmpDir)
	if cache == nil {
		t.Fatal("NewCache returned nil")
	}

	if cache.baseDir != tmpDir {
		t.Errorf("baseDir = %q, want %q", cache.baseDir, tmpDir)
	}

	if cache.ttl != 24*time.Hour {
		t.Errorf("default ttl = %v, want 24h", cache.ttl)
	}

	// Verify cache.json path
	expectedMetaPath := filepath.Join(tmpDir, "cache.json")
	if cache.metaPath != expectedMetaPath {
		t.Errorf("metaPath = %q, want %q", cache.metaPath, expectedMetaPath)
	}
}

// TestCacheSetTTL tests TTL modification
func TestCacheSetTTL(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	cache.SetTTL(1 * time.Hour)

	cache.mu.RLock()
	if cache.ttl != 1*time.Hour {
		t.Errorf("ttl = %v, want 1h", cache.ttl)
	}
	cache.mu.RUnlock()
}

// TestCacheGetRepoPath tests repo path generation
func TestCacheGetRepoPath(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	path := cache.GetRepoPath("test-source")
	expected := filepath.Join(tmpDir, "test-source")

	if path != expected {
		t.Errorf("GetRepoPath = %q, want %q", path, expected)
	}
}

// TestCacheNeedsUpdate tests update detection
func TestCacheNeedsUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Non-existent source needs update
	if !cache.NeedsUpdate("nonexistent") {
		t.Error("non-existent source should need update")
	}

	// Mark as updated
	cache.MarkUpdated("test-source")

	// Now should not need update
	if cache.NeedsUpdate("test-source") {
		t.Error("just-updated source should not need update")
	}

	// With short TTL, should need update
	cache.SetTTL(0)
	if !cache.NeedsUpdate("test-source") {
		t.Error("source with expired TTL should need update")
	}
}

// TestCacheMarkUpdated tests marking sources as updated
func TestCacheMarkUpdated(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	before := time.Now()
	cache.MarkUpdated("test-source")
	after := time.Now()

	lastUpdated := cache.GetLastUpdated("test-source")
	if lastUpdated.Before(before) || lastUpdated.After(after) {
		t.Errorf("LastUpdated = %v, should be between %v and %v", lastUpdated, before, after)
	}

	// Verify saved to disk
	cache2 := NewCache(tmpDir)
	lastUpdated2 := cache2.GetLastUpdated("test-source")
	if lastUpdated2.Before(before) {
		t.Error("metadata should be persisted to disk")
	}
}

// TestCacheCommit tests commit storage
func TestCacheCommit(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	cache.SetCommit("test-source", "abc123def456")
	commit := cache.GetCommit("test-source")

	if commit != "abc123def456" {
		t.Errorf("GetCommit = %q, want 'abc123def456'", commit)
	}

	// Verify persistence
	cache2 := NewCache(tmpDir)
	commit2 := cache2.GetCommit("test-source")
	if commit2 != "abc123def456" {
		t.Errorf("persisted commit = %q, want 'abc123def456'", commit2)
	}
}

// TestCacheGetMetadata tests metadata retrieval
func TestCacheGetMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	cache.MarkUpdated("source1")
	cache.SetCommit("source1", "commit1")
	cache.MarkUpdated("source2")
	cache.SetCommit("source2", "commit2")

	metadata := cache.GetMetadata()

	if len(metadata) != 2 {
		t.Errorf("metadata length = %d, want 2", len(metadata))
	}

	if metadata["source1"].Commit != "commit1" {
		t.Errorf("source1 commit = %q, want 'commit1'", metadata["source1"].Commit)
	}

	// Verify it's a copy (modifying doesn't affect original)
	metadata["source3"] = CacheMetadata{Name: "source3"}
	if len(cache.GetMetadata()) != 2 {
		t.Error("GetMetadata should return a copy")
	}
}

// TestCacheClear tests clearing a single source
func TestCacheClear(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Create a cached repo directory
	repoPath := cache.GetRepoPath("test-source")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	// Add metadata
	cache.MarkUpdated("test-source")
	cache.SetCommit("test-source", "abc123")

	// Clear
	if err := cache.Clear("test-source"); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// Verify metadata cleared
	if cache.GetCommit("test-source") != "" {
		t.Error("commit should be cleared")
	}

	// Verify directory removed
	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		t.Error("repo directory should be removed")
	}
}

// TestCacheClearAll tests clearing all cached sources
func TestCacheClearAll(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Create multiple cached repos
	for i := 0; i < 3; i++ {
		name := filepath.Join(tmpDir, "source"+string(rune('0'+i)))
		if err := os.MkdirAll(name, 0o755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		cache.MarkUpdated("source" + string(rune('0'+i)))
	}

	// Clear all
	if err := cache.ClearAll(); err != nil {
		t.Fatalf("ClearAll failed: %v", err)
	}

	// Verify all metadata cleared
	metadata := cache.GetMetadata()
	if len(metadata) != 0 {
		t.Errorf("metadata length = %d, want 0", len(metadata))
	}

	// cache.json should still exist
	if _, err := os.Stat(cache.metaPath); os.IsNotExist(err) {
		t.Error("cache.json should still exist")
	}
}

// TestCacheGetSize tests size calculation
func TestCacheGetSize(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Create some files
	testFile := filepath.Join(tmpDir, "testfile")
	content := []byte("test content here") // 17 bytes
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	size, err := cache.GetSize()
	if err != nil {
		t.Fatalf("GetSize failed: %v", err)
	}

	// Should be at least 17 bytes (plus cache.json if it exists)
	if size < 17 {
		t.Errorf("size = %d, want >= 17", size)
	}
}

// TestCacheExists tests existence checking
func TestCacheExists(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Non-existent
	if cache.Exists("nonexistent") {
		t.Error("non-existent source should not exist")
	}

	// Create directory
	repoPath := cache.GetRepoPath("test-source")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}

	// Now should exist
	if !cache.Exists("test-source") {
		t.Error("created source should exist")
	}
}

// TestCacheIsCached tests cached status checking
func TestCacheIsCached(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Non-existent: not cached
	if cache.IsCached("nonexistent") {
		t.Error("non-existent source should not be cached")
	}

	// Create directory and mark updated
	repoPath := cache.GetRepoPath("test-source")
	if err := os.MkdirAll(repoPath, 0o755); err != nil {
		t.Fatalf("failed to create repo dir: %v", err)
	}
	cache.MarkUpdated("test-source")

	// Now should be cached
	if !cache.IsCached("test-source") {
		t.Error("updated source should be cached")
	}

	// Expire it
	cache.ForceExpire("test-source")
	if cache.IsCached("test-source") {
		t.Error("expired source should not be cached")
	}
}

// TestCacheForceExpire tests forced expiration
func TestCacheForceExpire(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	cache.MarkUpdated("test-source")

	// Should not need update
	if cache.NeedsUpdate("test-source") {
		t.Error("should not need update after MarkUpdated")
	}

	// Force expire
	cache.ForceExpire("test-source")

	// Now should need update
	if !cache.NeedsUpdate("test-source") {
		t.Error("should need update after ForceExpire")
	}
}

// TestCacheGetCachedSources tests listing cached sources
func TestCacheGetCachedSources(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Initially empty
	sources := cache.GetCachedSources()
	if len(sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(sources))
	}

	// Add some sources
	cache.MarkUpdated("source1")
	cache.MarkUpdated("source2")
	cache.MarkUpdated("source3")

	sources = cache.GetCachedSources()
	if len(sources) != 3 {
		t.Errorf("expected 3 sources, got %d", len(sources))
	}

	// Verify all sources are present
	sourceMap := make(map[string]bool)
	for _, s := range sources {
		sourceMap[s] = true
	}

	for _, expected := range []string{"source1", "source2", "source3"} {
		if !sourceMap[expected] {
			t.Errorf("missing source: %s", expected)
		}
	}
}

// TestCacheMetadataStruct tests CacheMetadata fields
func TestCacheMetadataStruct(t *testing.T) {
	meta := CacheMetadata{
		Name:        "test-source",
		URL:         "https://github.com/example/repo",
		Branch:      "main",
		Commit:      "abc123",
		LastUpdated: time.Now(),
	}

	if meta.Name != "test-source" {
		t.Errorf("Name = %q, want 'test-source'", meta.Name)
	}

	if meta.URL != "https://github.com/example/repo" {
		t.Errorf("URL = %q, want git URL", meta.URL)
	}

	if meta.Branch != "main" {
		t.Errorf("Branch = %q, want 'main'", meta.Branch)
	}
}

// TestCacheLoadMetadataCorrupt tests handling of corrupt metadata
func TestCacheLoadMetadataCorrupt(t *testing.T) {
	tmpDir := t.TempDir()

	// Write corrupt JSON
	metaPath := filepath.Join(tmpDir, "cache.json")
	if err := os.WriteFile(metaPath, []byte("not valid json{"), 0o644); err != nil {
		t.Fatalf("failed to write corrupt file: %v", err)
	}

	// Should not panic, just log warning and continue
	cache := NewCache(tmpDir)
	if cache == nil {
		t.Fatal("NewCache should not return nil even with corrupt metadata")
	}

	// Should have empty metadata
	if len(cache.GetMetadata()) != 0 {
		t.Error("corrupt metadata should result in empty metadata map")
	}
}

// TestCacheLoadMetadataMissing tests handling of missing metadata file
func TestCacheLoadMetadataMissing(t *testing.T) {
	tmpDir := t.TempDir()

	// Just create cache without any existing metadata file
	cache := NewCache(tmpDir)
	if cache == nil {
		t.Fatal("NewCache should not return nil with missing metadata")
	}

	// Should have empty metadata
	if len(cache.GetMetadata()) != 0 {
		t.Error("missing metadata should result in empty metadata map")
	}
}
