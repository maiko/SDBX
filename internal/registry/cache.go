package registry

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Cache manages caching of Git sources
type Cache struct {
	baseDir  string
	ttl      time.Duration
	metadata map[string]CacheMetadata
	mu       sync.RWMutex
	metaPath string
}

// CacheMetadata stores metadata about cached sources
type CacheMetadata struct {
	Name        string    `json:"name"`
	URL         string    `json:"url,omitempty"`
	Branch      string    `json:"branch,omitempty"`
	Commit      string    `json:"commit,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

// NewCache creates a new Cache
func NewCache(baseDir string) *Cache {
	c := &Cache{
		baseDir:  baseDir,
		ttl:      24 * time.Hour,
		metadata: make(map[string]CacheMetadata),
		metaPath: filepath.Join(baseDir, "cache.json"),
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		log.Printf("Warning: failed to create cache directory: %v", err)
	}

	// Load existing metadata
	c.loadMetadata()

	return c
}

// SetTTL sets the cache TTL
func (c *Cache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ttl = ttl
}

// GetRepoPath returns the path where a repo should be cached
func (c *Cache) GetRepoPath(sourceName string) string {
	return filepath.Join(c.baseDir, sourceName)
}

// NeedsUpdate checks if a source needs to be updated
func (c *Cache) NeedsUpdate(sourceName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	meta, exists := c.metadata[sourceName]
	if !exists {
		return true
	}

	return time.Since(meta.LastUpdated) > c.ttl
}

// MarkUpdated marks a source as updated
func (c *Cache) MarkUpdated(sourceName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	meta := c.metadata[sourceName]
	meta.Name = sourceName
	meta.LastUpdated = time.Now()
	c.metadata[sourceName] = meta

	c.saveMetadata()
}

// SetCommit stores the commit hash for a source
func (c *Cache) SetCommit(sourceName, commit string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	meta := c.metadata[sourceName]
	meta.Commit = commit
	c.metadata[sourceName] = meta

	c.saveMetadata()
}

// GetCommit returns the cached commit hash for a source
func (c *Cache) GetCommit(sourceName string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.metadata[sourceName].Commit
}

// GetLastUpdated returns when a source was last updated
func (c *Cache) GetLastUpdated(sourceName string) time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.metadata[sourceName].LastUpdated
}

// GetMetadata returns all cache metadata
func (c *Cache) GetMetadata() map[string]CacheMetadata {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy
	result := make(map[string]CacheMetadata)
	for k, v := range c.metadata {
		result[k] = v
	}
	return result
}

// Clear clears the cache for a specific source
func (c *Cache) Clear(sourceName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove from metadata
	delete(c.metadata, sourceName)
	c.saveMetadata()

	// Remove cached files
	repoPath := c.GetRepoPath(sourceName)
	return os.RemoveAll(repoPath)
}

// ClearAll clears all cached sources
func (c *Cache) ClearAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metadata = make(map[string]CacheMetadata)
	c.saveMetadata()

	// Remove all files except metadata
	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Name() == "cache.json" {
			continue
		}
		path := filepath.Join(c.baseDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}

	return nil
}

// GetSize returns the total size of the cache in bytes
func (c *Cache) GetSize() (int64, error) {
	var size int64

	err := filepath.WalkDir(c.baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		return nil
	})

	return size, err
}

// loadMetadata loads cache metadata from disk
func (c *Cache) loadMetadata() {
	data, err := os.ReadFile(c.metaPath)
	if err != nil {
		// File not existing is normal on first run
		if !os.IsNotExist(err) {
			log.Printf("Warning: failed to read cache metadata: %v", err)
		}
		return
	}

	var metadata map[string]CacheMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		log.Printf("Warning: failed to parse cache metadata: %v", err)
		return
	}

	c.metadata = metadata
}

// saveMetadata saves cache metadata to disk
func (c *Cache) saveMetadata() {
	data, err := json.MarshalIndent(c.metadata, "", "  ")
	if err != nil {
		log.Printf("Warning: failed to marshal cache metadata: %v", err)
		return
	}

	if err := os.WriteFile(c.metaPath, data, 0o644); err != nil {
		log.Printf("Warning: failed to save cache metadata: %v", err)
	}
}

// Exists checks if a source is cached
func (c *Cache) Exists(sourceName string) bool {
	repoPath := c.GetRepoPath(sourceName)
	_, err := os.Stat(repoPath)
	return err == nil
}

// IsCached checks if a source is cached and not expired
func (c *Cache) IsCached(sourceName string) bool {
	if !c.Exists(sourceName) {
		return false
	}
	return !c.NeedsUpdate(sourceName)
}

// ForceExpire forces a source to be marked as needing update
func (c *Cache) ForceExpire(sourceName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	meta := c.metadata[sourceName]
	meta.LastUpdated = time.Time{} // Zero time
	c.metadata[sourceName] = meta

	c.saveMetadata()
}

// GetCachedSources returns names of all cached sources
func (c *Cache) GetCachedSources() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var names []string
	for name := range c.metadata {
		names = append(names, name)
	}
	return names
}
