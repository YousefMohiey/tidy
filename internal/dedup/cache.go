package dedup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/YousefMohiey/tidy/internal/paths"
)

// CacheEntry stores the hash and metadata for a single file.
type CacheEntry struct {
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mod_time"`
	Hash     string    `json:"hash"`
	CachedAt time.Time `json:"cached_at"`
}

// HashCache persists file hashes across scans to avoid re-hashing unchanged files.
type HashCache struct {
	entries map[string]*CacheEntry
	path    string
	mu      sync.RWMutex
}

// NewHashCache creates or loads a hash cache from the default location.
func NewHashCache() *HashCache {
	cachePath := filepath.Join(paths.DataDir(), "hashcache.json")
	c := &HashCache{
		entries: make(map[string]*CacheEntry),
		path:    cachePath,
	}
	_ = c.Load()
	return c
}

// Load reads the cache from disk.
func (c *HashCache) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var entries []CacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	for i := range entries {
		e := &entries[i]
		c.entries[e.Path] = e
	}
	return nil
}

// Save writes the cache to disk.
func (c *HashCache) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return err
	}

	entries := make([]CacheEntry, 0, len(c.entries))
	for _, e := range c.entries {
		entries = append(entries, *e)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0o644)
}

// Get returns a cached hash if the file is unchanged (same size + modtime).
func (c *HashCache) Get(path string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[path]
	if !ok {
		return "", false
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}

	if info.Size() != entry.Size || !info.ModTime().Equal(entry.ModTime) {
		return "", false
	}

	return entry.Hash, true
}

// Set stores a hash for a file.
func (c *HashCache) Set(path string, size int64, modTime time.Time, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[path] = &CacheEntry{
		Path:     path,
		Size:     size,
		ModTime:  modTime,
		Hash:     hash,
		CachedAt: time.Now(),
	}
}

// Prune removes entries for files that no longer exist.
func (c *HashCache) Prune() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for path := range c.entries {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			delete(c.entries, path)
			removed++
		}
	}
	return removed
}

// Size returns the number of cached entries.
func (c *HashCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}
