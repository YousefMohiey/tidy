// Package dedup provides duplicate file detection by content hashing.
// It uses a size-first optimization to avoid hashing files with unique sizes,
// then applies SHA256 streaming hashes only to size-matched candidates.
package dedup

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DuplicateGroup represents a set of files with identical content.
type DuplicateGroup struct {
	Hash  string   `json:"hash"`  // SHA256 hex digest
	Size  int64    `json:"size"`  // file size in bytes
	Files []string `json:"files"` // absolute paths
}

// ScanResult holds the outcome of a duplicate scan.
type ScanResult struct {
	TotalFiles      int              `json:"total_files"`
	UniqueFiles     int              `json:"unique_files"`
	DuplicateGroups []DuplicateGroup `json:"duplicate_groups"`
	WastedBytes     int64            `json:"wasted_bytes"` // total bytes that could be reclaimed
	ScannedDirs     []string         `json:"scanned_dirs"`
}

// Scanner finds duplicate files across one or more directories.
type Scanner struct {
	// MinSize is the minimum file size to consider (skip empty files).
	// Default: 1 byte.
	MinSize int64
}

// NewScanner creates a Scanner with default settings.
func NewScanner() *Scanner {
	return &Scanner{
		MinSize: 1,
	}
}

// Scan walks the given directories (recursively), groups files by size first
// (optimization: files with unique sizes cannot be duplicates), then hashes
// only size-matched files with SHA256.
// Returns a ScanResult with all duplicate groups found.
func (s *Scanner) Scan(dirs ...string) (*ScanResult, error) {
	if len(dirs) == 0 {
		return nil, fmt.Errorf("scan: no directories provided")
	}

	minSize := s.MinSize
	if minSize <= 0 {
		minSize = 1
	}

	// Resolve and deduplicate input directories
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

	// Phase 1: Walk all directories, group files by size
	sizeGroups := make(map[int64][]string)
	totalFiles := 0

	for _, dir := range scannedDirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// Skip entries we can't access
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			name := d.Name()

			// Skip hidden files and directories
			if strings.HasPrefix(name, ".") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip symlinks
			if d.Type()&fs.ModeSymlink != 0 {
				return nil
			}

			// Skip directories (but continue recursing into them)
			if d.IsDir() {
				return nil
			}

			// Get file info for size
			info, err := d.Info()
			if err != nil {
				return nil // skip unreadable entries
			}

			size := info.Size()
			if size < minSize {
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

	// Phase 2: Filter to only sizes with multiple files (potential duplicates)
	// and hash those files
	hashGroups := make(map[string]*DuplicateGroup)

	for size, paths := range sizeGroups {
		if len(paths) < 2 {
			continue // unique size, cannot be duplicate
		}

		for _, path := range paths {
			hash, err := hashFile(path)
			if err != nil {
				continue // skip unreadable files
			}

			key := fmt.Sprintf("%s:%d", hash, size)
			group, exists := hashGroups[key]
			if !exists {
				group = &DuplicateGroup{
					Hash:  hash,
					Size:  size,
					Files: make([]string, 0),
				}
				hashGroups[key] = group
			}
			group.Files = append(group.Files, path)
		}
	}

	// Phase 3: Build result — only groups with 2+ files are duplicates
	dupGroups := make([]DuplicateGroup, 0)
	filesInDupGroups := 0

	for _, group := range hashGroups {
		if len(group.Files) < 2 {
			continue
		}

		// Sort files alphabetically within each group
		sort.Strings(group.Files)

		filesInDupGroups += len(group.Files)
		dupGroups = append(dupGroups, *group)
	}

	// Sort groups by wasted bytes descending (biggest duplicates first)
	sort.Slice(dupGroups, func(i, j int) bool {
		wasteI := (int64(len(dupGroups[i].Files)) - 1) * dupGroups[i].Size
		wasteJ := (int64(len(dupGroups[j].Files)) - 1) * dupGroups[j].Size
		if wasteI != wasteJ {
			return wasteI > wasteJ
		}
		// Tie-break by first file path for deterministic output
		return dupGroups[i].Files[0] < dupGroups[j].Files[0]
	})

	// Calculate total wasted bytes
	var wastedBytes int64
	for _, g := range dupGroups {
		wastedBytes += (int64(len(g.Files)) - 1) * g.Size
	}

	// Sort scanned dirs for deterministic output
	sort.Strings(scannedDirs)

	return &ScanResult{
		TotalFiles:      totalFiles,
		UniqueFiles:     totalFiles - filesInDupGroups,
		DuplicateGroups: dupGroups,
		WastedBytes:     wastedBytes,
		ScannedDirs:     scannedDirs,
	}, nil
}

// hashFile computes the SHA256 hex digest of a file using streaming I/O.
// The file is never loaded entirely into memory.
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
// For each group of N identical files, (N-1) * size is wasted.
func (r *ScanResult) WastedBytesTotal() int64 {
	var total int64
	for _, g := range r.DuplicateGroups {
		total += (int64(len(g.Files)) - 1) * g.Size
	}
	return total
}

// FormatSize returns a human-readable file size string using binary units.
// Examples: "0 B", "512 B", "1.5 KB", "23 MB", "1.2 GB"
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
		val := float64(bytes) / float64(tb)
		return formatDecimal(val) + " TB"
	case bytes >= int64(gb):
		val := float64(bytes) / float64(gb)
		return formatDecimal(val) + " GB"
	case bytes >= int64(mb):
		val := float64(bytes) / float64(mb)
		return formatDecimal(val) + " MB"
	case bytes >= int64(kb):
		val := float64(bytes) / float64(kb)
		return formatDecimal(val) + " KB"
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatDecimal formats a float value: one decimal place if < 10, none if >= 10.
func formatDecimal(val float64) string {
	if val < 10.0 {
		return fmt.Sprintf("%.1f", val)
	}
	return fmt.Sprintf("%.0f", val)
}
