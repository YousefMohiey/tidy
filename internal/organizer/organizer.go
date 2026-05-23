package organizer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/verhafter/tidy/internal/config"
	"github.com/verhafter/tidy/internal/detector"
	"github.com/verhafter/tidy/internal/rules"
)

// Options controls organizer behavior.
type Options struct {
	DryRun     bool
	JournalDir string // where to save journal files, default ~/.local/share/tidy/
}

// Result summarizes the outcome of an Organize operation.
type Result struct {
	FilesProcessed int
	FilesMoved     int
	FilesSkipped   int
	Errors         []error
	DryRun         bool
	Moves          []MoveRecord // populated even in dry-run for preview
}

// Organizer orchestrates file organization according to configured rules.
type Organizer struct {
	engine  *rules.Engine
	options Options
}

// New creates an Organizer with the given configuration and options.
func New(cfg *config.Config, opts Options) *Organizer {
	if opts.JournalDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			opts.JournalDir = filepath.Join(home, ".local", "share", "tidy")
		} else {
			opts.JournalDir = ".local/share/tidy"
		}
	}

	return &Organizer{
		engine:  rules.NewEngine(cfg),
		options: opts,
	}
}

// Organize scans the directory and organizes files according to rules.
// It does NOT recurse into subdirectories — only top-level files.
// Skips hidden files (starting with .), symlinks, and the journal directory itself.
func (o *Organizer) Organize(dir string) (*Result, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("organize: failed to read directory %q: %w", dir, err)
	}

	journal := NewJournal(dir)
	result := &Result{
		DryRun: o.options.DryRun,
		Moves:  []MoveRecord{},
	}

	// Resolve journal directory for comparison
	journalDirAbs, _ := filepath.Abs(o.options.JournalDir)
	targetDirAbs, _ := filepath.Abs(dir)

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files/directories
		if strings.HasPrefix(name, ".") {
			result.FilesSkipped++
			continue
		}

		// Skip directories (no recursion)
		if entry.IsDir() {
			result.FilesSkipped++
			continue
		}

		srcPath := filepath.Join(dir, name)

		// Skip symlinks
		if info, err := os.Lstat(srcPath); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				result.FilesSkipped++
				continue
			}
		}

		// Skip if this is the journal directory itself
		srcAbs, _ := filepath.Abs(srcPath)
		if srcAbs == journalDirAbs && strings.HasPrefix(journalDirAbs, targetDirAbs) {
			result.FilesSkipped++
			continue
		}

		result.FilesProcessed++

		// Detect MIME type
		mimeType, err := detector.Detect(srcPath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("detect %q: %w", name, err))
			result.FilesSkipped++
			continue
		}

		// Match against rules
		match := o.engine.Match(name, mimeType)
		category := "Other"
		destSubdir := "Other"
		if match != nil {
			category = match.Category
			destSubdir = match.Destination
		}

		// Build destination path
		destDir := filepath.Join(dir, destSubdir)
		destPath := o.resolveDestination(srcPath, destDir)

		record := MoveRecord{
			Source:      srcPath,
			Destination: destPath,
		}
		result.Moves = append(result.Moves, record)

		if o.options.DryRun {
			result.FilesMoved++
			continue
		}

		// Create destination directory if needed
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("mkdir %q for %q: %w", destDir, category, err))
			result.FilesSkipped++
			continue
		}

		// Move the file
		if err := os.Rename(srcPath, destPath); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("move %q → %q: %w", srcPath, destPath, err))
			result.FilesSkipped++
			continue
		}

		journal.Record(srcPath, destPath)
		result.FilesMoved++
	}

	// Save journal if we made changes
	if !o.options.DryRun && len(journal.Operations) > 0 {
		journalPath := filepath.Join(o.options.JournalDir, "journal.json")
		if err := journal.Save(journalPath); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("save journal: %w", err))
		}
	}

	return result, nil
}

// resolveDestination determines the full destination path for a file.
// If a file already exists at the destination, appends a counter:
//
//	photo.jpg → photo_1.jpg → photo_2.jpg
func (o *Organizer) resolveDestination(src, destDir string) string {
	base := filepath.Base(src)
	candidate := filepath.Join(destDir, base)

	// Fast path: no conflict
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}

	// Split filename and extension
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Try numbered variants
	for i := 1; ; i++ {
		candidate = filepath.Join(destDir, fmt.Sprintf("%s_%d%s", name, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
