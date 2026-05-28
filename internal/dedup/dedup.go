// Package dedup provides duplicate file detection by content hashing.
// It uses a size-first optimization to avoid hashing files with unique sizes,
// then applies SHA256 streaming hashes only to size-matched candidates.
// A persistent hash cache avoids re-hashing unchanged files on subsequent scans.
package dedup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
)

// DuplicateGroup represents a set of files with identical content.
type DuplicateGroup struct {
	Hash  string   `json:"hash"`
	Size  int64    `json:"size"`
	Files []string `json:"files"`
}

// ScanResult holds the outcome of a duplicate scan.
type ScanResult struct {
	TotalFiles      int              `json:"total_files"`
	UniqueFiles     int              `json:"unique_files"`
	DuplicateGroups []DuplicateGroup `json:"duplicate_groups"`
	WastedBytes     int64            `json:"wasted_bytes"`
	ScannedDirs     []string         `json:"scanned_dirs"`
	CacheHits       int              `json:"cache_hits"`
	CacheMisses     int              `json:"cache_misses"`
}

// Progress reports the current state of a dedup scan.
type Progress struct {
	FilesTotal     int
	FilesScanned   int
	FilesHashed    int
	CacheHits      int
	CurrentFile    string
	Status         string
}

// Scanner finds duplicate files across one or more directories.
type Scanner struct {
	MinSize   int64
	MaxSize   int64
	MaxDepth  int
	Cache     *HashCache
	OnProgress func(Progress)
}

// NewScanner creates a Scanner with default settings.
func NewScanner() *Scanner {
	return &Scanner{
		MinSize:  1,
		MaxSize:  1073741824,
		MaxDepth: 0,
		Cache:    NewHashCache(),
	}
}

func (s *Scanner) reportProgress(p Progress) {
	if s.OnProgress != nil {
		s.OnProgress(p)
	}
}

// Scan walks the given directories, groups files by size, then hashes
// only size-matched files with SHA256. Uses cache to skip unchanged files.
func (s *Scanner) Scan(dirs ...string) (*ScanResult, error) {
	if len(dirs) == 0 {
		return nil, fmt.Errorf("scan: no directories provided")
	}

	minSize := s.MinSize
	if minSize <= 0 {
		minSize = 1
	}

	scannedDirs := make([]string, 0, len(dirs))
	seenDirs := make(map[string]bool)
	for _, dir := range dirs {
		abs, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			continue
		}
		if !seenDirs[abs] {
			seenDirs[abs] = true
			scannedDirs = append(scannedDirs, abs)
		}
	}

	if len(scannedDirs) == 0 {
		return nil, fmt.Errorf("scan: no valid directories found")
	}

	sizeGroups := make(map[int64][]string)
	totalFiles := 0

	for _, dir := range scannedDirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			name := d.Name()
			if strings.HasPrefix(name, ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			skipDirs := map[string]bool{
				"node_modules": true, ".git": true, "__pycache__": true,
				"venv": true, ".venv": true, "vendor": true,
				".cargo": true, "target": true, "obj": true, "bin": true,
			}
			if d.IsDir() && skipDirs[strings.ToLower(name)] {
				return filepath.SkipDir
			}

			if d.Type()&fs.ModeSymlink != 0 {
				return nil
			}
			if d.IsDir() {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}

			size := info.Size()
			if size < minSize {
				return nil
			}
			if s.MaxSize > 0 && size > s.MaxSize {
				return nil
			}

			totalFiles++
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}
			sizeGroups[size] = append(sizeGroups[size], absPath)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan: walk %q: %w", dir, err)
		}
	}

	type hashTask struct {
		path string
		size int64
	}
	var tasks []hashTask

	for size, paths := range sizeGroups {
		if len(paths) < 2 {
			continue
		}
		for _, path := range paths {
			tasks = append(tasks, hashTask{path: path, size: size})
		}
	}

	cacheHits := 0
	cacheMisses := 0
	hashGroups := make(map[string]*DuplicateGroup)
	var mu sync.Mutex
	var wg sync.WaitGroup

	workers := runtime.NumCPU()
	if workers > 4 {
		workers = 4
	}
	if workers < 1 {
		workers = 1
	}

	ch := make(chan hashTask, workers)
	scanned := 0
	var scanMu sync.Mutex

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range ch {
				scanMu.Lock()
				scanned++
				sc := scanned
				scanMu.Unlock()

				s.reportProgress(Progress{
					FilesTotal:   len(tasks),
					FilesScanned: sc,
					CacheHits:    cacheHits,
					CurrentFile:  filepath.Base(t.path),
					Status:       "hashing",
				})

				info, err := os.Stat(t.path)
				if err != nil {
					continue
				}

				var hash string
				if s.Cache != nil {
					if cached, ok := s.Cache.Get(t.path); ok {
						hash = cached
						mu.Lock()
						cacheHits++
						mu.Unlock()
					}
				}

				if hash == "" {
					h, err := hashFile(t.path)
					if err != nil {
						continue
					}
					hash = h
					if s.Cache != nil {
						s.Cache.Set(t.path, info.Size(), info.ModTime(), hash)
					}
					mu.Lock()
					cacheMisses++
					mu.Unlock()
				}

				key := fmt.Sprintf("%s:%d", hash, t.size)
				mu.Lock()
				group, exists := hashGroups[key]
				if !exists {
					group = &DuplicateGroup{
						Hash:  hash,
						Size:  t.size,
						Files: make([]string, 0),
					}
					hashGroups[key] = group
				}
				group.Files = append(group.Files, t.path)
				mu.Unlock()
			}
		}()
	}

	for _, t := range tasks {
		ch <- t
	}
	close(ch)
	wg.Wait()

	if s.Cache != nil {
		_ = s.Cache.Save()
	}

	dupGroups := make([]DuplicateGroup, 0)
	filesInDupGroups := 0

	for _, group := range hashGroups {
		if len(group.Files) < 2 {
			continue
		}
		sort.Strings(group.Files)
		filesInDupGroups += len(group.Files)
		dupGroups = append(dupGroups, *group)
	}

	sort.Slice(dupGroups, func(i, j int) bool {
		wasteI := (int64(len(dupGroups[i].Files)) - 1) * dupGroups[i].Size
		wasteJ := (int64(len(dupGroups[j].Files)) - 1) * dupGroups[j].Size
		if wasteI != wasteJ {
			return wasteI > wasteJ
		}
		return dupGroups[i].Files[0] < dupGroups[j].Files[0]
	})

	var wastedBytes int64
	for _, g := range dupGroups {
		wastedBytes += (int64(len(g.Files)) - 1) * g.Size
	}

	sort.Strings(scannedDirs)

	return &ScanResult{
		TotalFiles:      totalFiles,
		UniqueFiles:     totalFiles - filesInDupGroups,
		DuplicateGroups: dupGroups,
		WastedBytes:     wastedBytes,
		ScannedDirs:     scannedDirs,
		CacheHits:       cacheHits,
		CacheMisses:     cacheMisses,
	}, nil
}

// DeleteGroup removes all files in a duplicate group except the first one.
func (r *ScanResult) DeleteGroup(index int) (int64, error) {
	if index < 0 || index >= len(r.DuplicateGroups) {
		return 0, fmt.Errorf("delete: invalid group index %d", index)
	}
	g := &r.DuplicateGroups[index]
	if len(g.Files) < 2 {
		return 0, nil
	}

	var deleted int64
	for i := 1; i < len(g.Files); i++ {
		if err := os.Remove(g.Files[i]); err == nil {
			deleted += g.Size
		}
	}

	g.Files = g.Files[:1]
	r.WastedBytes -= deleted
	return deleted, nil
}

// KeepOne removes all files in a group except the one at keepIndex.
func (r *ScanResult) KeepOne(groupIndex, keepIndex int) (int64, error) {
	if groupIndex < 0 || groupIndex >= len(r.DuplicateGroups) {
		return 0, fmt.Errorf("keep: invalid group index %d", groupIndex)
	}
	g := &r.DuplicateGroups[groupIndex]
	if keepIndex < 0 || keepIndex >= len(g.Files) {
		return 0, fmt.Errorf("keep: invalid file index %d", keepIndex)
	}

	var deleted int64
	kept := g.Files[keepIndex]
	remaining := []string{kept}

	for i, f := range g.Files {
		if i == keepIndex {
			continue
		}
		if err := os.Remove(f); err == nil {
			deleted += g.Size
		}
	}

	g.Files = remaining
	r.WastedBytes -= deleted
	return deleted, nil
}

// hashFile computes the SHA256 hex digest of a file using streaming I/O.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// WastedBytesTotal calculates total reclaimable bytes across all groups.
func (r *ScanResult) WastedBytesTotal() int64 {
	var total int64
	for _, g := range r.DuplicateGroups {
		total += (int64(len(g.Files)) - 1) * g.Size
	}
	return total
}

// FormatSize returns a human-readable file size string using binary units.
func FormatSize(bytes int64) string {
	if bytes < 0 {
		bytes = 0
	}
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
		tb = 1024 * gb
	)
	switch {
	case bytes >= int64(tb):
		return formatDecimal(float64(bytes)/float64(tb)) + " TB"
	case bytes >= int64(gb):
		return formatDecimal(float64(bytes)/float64(gb)) + " GB"
	case bytes >= int64(mb):
		return formatDecimal(float64(bytes)/float64(mb)) + " MB"
	case bytes >= int64(kb):
		return formatDecimal(float64(bytes)/float64(kb)) + " KB"
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatDecimal(val float64) string {
	if val < 10.0 {
		return fmt.Sprintf("%.1f", val)
	}
	return fmt.Sprintf("%.0f", val)
}
