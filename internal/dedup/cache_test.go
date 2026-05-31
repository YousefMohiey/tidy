package dedup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestCache(t *testing.T) *HashCache {
	t.Helper()
	return &HashCache{
		entries: make(map[string]*CacheEntry),
		path:    filepath.Join(t.TempDir(), "cache.json"),
	}
}

func TestCacheSetAndGet(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat test file: %v", err)
	}

	c := newTestCache(t)
	c.Set(testFile, info.Size(), info.ModTime(), "abc123")

	hash, ok := c.Get(testFile)
	if !ok {
		t.Fatal("Get returned false, want true")
	}
	if hash != "abc123" {
		t.Errorf("Get hash = %q, want %q", hash, "abc123")
	}
}

func TestCacheGetMismatchedSize(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat test file: %v", err)
	}

	c := newTestCache(t)
	// Set with wrong size
	c.Set(testFile, info.Size()+100, info.ModTime(), "abc123")

	_, ok := c.Get(testFile)
	if ok {
		t.Error("Get returned true for mismatched size, want false")
	}
}

func TestCacheGetMismatchedModTime(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat test file: %v", err)
	}

	c := newTestCache(t)
	// Set with wrong modtime
	c.Set(testFile, info.Size(), info.ModTime().Add(-time.Hour), "abc123")

	_, ok := c.Get(testFile)
	if ok {
		t.Error("Get returned true for mismatched modtime, want false")
	}
}

func TestCacheGetNonexistentFile(t *testing.T) {
	c := newTestCache(t)
	_, ok := c.Get(filepath.Join(t.TempDir(), "nonexistent.txt"))
	if ok {
		t.Error("Get returned true for nonexistent file, want false")
	}
}

func TestCacheSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat test file: %v", err)
	}

	// Create and populate first cache
	c1 := newTestCache(t)
	c1.Set(testFile, info.Size(), info.ModTime(), "hash123")

	if err := c1.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Create second cache and load from same path
	c2 := &HashCache{
		entries: make(map[string]*CacheEntry),
		path:    c1.path,
	}
	if err := c2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify the entry was loaded
	hash, ok := c2.Get(testFile)
	if !ok {
		t.Fatal("Get returned false after Load, want true")
	}
	if hash != "hash123" {
		t.Errorf("Get hash after Load = %q, want %q", hash, "hash123")
	}
}

func TestCacheLoadNonexistentFile(t *testing.T) {
	c := newTestCache(t)
	// Load from nonexistent file should not error
	if err := c.Load(); err != nil {
		t.Errorf("Load from nonexistent file returned error: %v", err)
	}
	if c.Size() != 0 {
		t.Errorf("Size after loading nonexistent file = %d, want 0", c.Size())
	}
}

func TestCachePrune(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0o644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat test file: %v", err)
	}

	c := newTestCache(t)
	c.Set(testFile, info.Size(), info.ModTime(), "hash1")

	// Add entry for file that will be deleted
	deletedFile := filepath.Join(dir, "deleted.txt")
	if err := os.WriteFile(deletedFile, []byte("temp"), 0o644); err != nil {
		t.Fatalf("failed to create deleted file: %v", err)
	}
	deletedInfo, err := os.Stat(deletedFile)
	if err != nil {
		t.Fatalf("failed to stat deleted file: %v", err)
	}
	c.Set(deletedFile, deletedInfo.Size(), deletedInfo.ModTime(), "hash2")

	if c.Size() != 2 {
		t.Errorf("Size before Prune = %d, want 2", c.Size())
	}

	// Delete one file
	if err := os.Remove(deletedFile); err != nil {
		t.Fatalf("failed to remove file: %v", err)
	}

	removed := c.Prune()
	if removed != 1 {
		t.Errorf("Prune removed = %d, want 1", removed)
	}

	if c.Size() != 1 {
		t.Errorf("Size after Prune = %d, want 1", c.Size())
	}

	// Verify the surviving entry is still accessible
	_, ok := c.Get(testFile)
	if !ok {
		t.Error("Get returned false for surviving entry after Prune")
	}
}

func TestCachePruneAllGone(t *testing.T) {
	c := newTestCache(t)
	// Add entries for nonexistent files
	c.Set(filepath.Join(t.TempDir(), "gone1.txt"), 10, time.Now(), "h1")
	c.Set(filepath.Join(t.TempDir(), "gone2.txt"), 20, time.Now(), "h2")

	if c.Size() != 2 {
		t.Errorf("Size before Prune = %d, want 2", c.Size())
	}

	removed := c.Prune()
	if removed != 2 {
		t.Errorf("Prune removed = %d, want 2", removed)
	}

	if c.Size() != 0 {
		t.Errorf("Size after Prune = %d, want 0", c.Size())
	}
}

func TestCacheSize(t *testing.T) {
	c := newTestCache(t)
	if c.Size() != 0 {
		t.Errorf("Size of new cache = %d, want 0", c.Size())
	}

	c.Set("path1", 10, time.Now(), "h1")
	c.Set("path2", 20, time.Now(), "h2")

	if c.Size() != 2 {
		t.Errorf("Size after 2 Sets = %d, want 2", c.Size())
	}

	// Overwrite existing entry
	c.Set("path1", 30, time.Now(), "h3")
	if c.Size() != 2 {
		t.Errorf("Size after overwrite = %d, want 2", c.Size())
	}
}
