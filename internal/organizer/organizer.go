package organizer

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/verhafter/tidy/internal/config"
	"github.com/verhafter/tidy/internal/detector"
	"github.com/verhafter/tidy/internal/paths"
	"github.com/verhafter/tidy/internal/rules"
)

// Options controls organizer behavior.
type Options struct {
	DryRun     bool
	JournalDir string
	OnProgress func(Progress)
}

// Progress reports the current state of an organize operation.
type Progress struct {
	FilesTotal     int
	FilesProcessed int
	FilesMoved     int
	FilesSkipped   int
	CurrentFile    string
	Status         string // e.g. "detecting", "moving", "done"
}

// Result summarizes the outcome of an Organize operation.
type Result struct {
	FilesProcessed int
	FilesMoved     int
	FilesSkipped   int
	Errors         []error
	DryRun         bool
	Moves          []MoveRecord
}

// TreePreview represents a hierarchical preview of what organize would do.
type TreePreview struct {
	SourceDir  string
	Categories map[string]*CategoryPreview
}

// CategoryPreview holds files for one destination category.
type CategoryPreview struct {
	Name  string
	Files []string
}

type fileCandidate struct {
	entry os.DirEntry
	name  string
	path  string
}

// Organizer orchestrates file organization according to configured rules.
type Organizer struct {
	engine  *rules.Engine
	options Options
}

// New creates an Organizer with the given configuration and options.
func New(cfg *config.Config, opts Options) *Organizer {
	if opts.JournalDir == "" {
		opts.JournalDir = paths.DataDir()
	}

	return &Organizer{
		engine:  rules.NewEngine(cfg),
		options: opts,
	}
}

func (o *Organizer) reportProgress(p Progress) {
	if o.options.OnProgress != nil {
		o.options.OnProgress(p)
	}
}

// Organize scans the directory and organizes files according to rules.
func (o *Organizer) Organize(dir string) (*Result, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("organize: failed to read directory %q: %w", dir, err)
	}

	// Collect candidate files first
	var candidates []fileCandidate

	journalDirAbs, _ := filepath.Abs(o.options.JournalDir)
	targetDirAbs, _ := filepath.Abs(dir)

	for _, entry := range entries {
		name := entry.Name()

		if strings.HasPrefix(name, ".") {
			continue
		}
		if strings.HasSuffix(strings.ToLower(name), ".lnk") {
			continue
		}
		if entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(dir, name)

		if info, err := os.Lstat(srcPath); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				continue
			}
		}

		srcAbs, _ := filepath.Abs(srcPath)
		if srcAbs == journalDirAbs && strings.HasPrefix(journalDirAbs, targetDirAbs) {
			continue
		}

		candidates = append(candidates, fileCandidate{entry: entry, name: name, path: srcPath})
	}

	total := len(candidates)
	journal := NewJournal(dir)
	result := &Result{
		DryRun: o.options.DryRun,
		Moves:  []MoveRecord{},
	}

	o.reportProgress(Progress{
		FilesTotal: total,
		Status:     "detecting",
	})

	// Process files with bounded parallelism
	var mu sync.Mutex
	var wg sync.WaitGroup

	workers := runtime.NumCPU()
	if workers > 4 {
		workers = 4
	}
	if workers < 1 {
		workers = 1
	}

	ch := make(chan fileCandidate, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for c := range ch {
				o.processFile(dir, c, journal, result, &mu, total)
			}
		}()
	}

	for _, c := range candidates {
		ch <- c
	}
	close(ch)
	wg.Wait()

	if !o.options.DryRun && len(journal.Operations) > 0 {
		journalPath := filepath.Join(o.options.JournalDir, "journal.json")
		if err := journal.Append(journalPath); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("save journal: %w", err))
		}
		_ = backupJournal(journalPath)
	}

	o.reportProgress(Progress{
		FilesTotal:     total,
		FilesProcessed: result.FilesProcessed,
		FilesMoved:     result.FilesMoved,
		FilesSkipped:   result.FilesSkipped,
		Status:         "done",
	})

	return result, nil
}

func (o *Organizer) processFile(dir string, c fileCandidate, journal *Journal, result *Result, mu *sync.Mutex, total int) {
	srcPath := c.path
	name := c.name

	mu.Lock()
	result.FilesProcessed++
	processed := result.FilesProcessed
	mu.Unlock()

	o.reportProgress(Progress{
		FilesTotal:     total,
		FilesProcessed: processed,
		FilesMoved:     result.FilesMoved,
		FilesSkipped:   result.FilesSkipped,
		CurrentFile:    name,
		Status:         "detecting",
	})

	mimeType, err := detector.Detect(srcPath)
	if err != nil {
		mu.Lock()
		result.Errors = append(result.Errors, fmt.Errorf("detect %q: %w", name, err))
		result.FilesSkipped++
		mu.Unlock()
		return
	}

	match := o.engine.Match(name, mimeType)
	category := "Other"
	destSubdir := "Other"
	if match != nil {
		category = match.Category
		destSubdir = match.Destination
	}

	destDir := filepath.Join(dir, destSubdir)
	destPath := o.resolveDestination(srcPath, destDir)

	record := MoveRecord{
		Source:      srcPath,
		Destination: destPath,
		Category:    category,
	}

	mu.Lock()
	result.Moves = append(result.Moves, record)
	mu.Unlock()

	if o.options.DryRun {
		mu.Lock()
		result.FilesMoved++
		mu.Unlock()
		return
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		mu.Lock()
		result.Errors = append(result.Errors, fmt.Errorf("mkdir %q for %q: %w", destDir, category, err))
		result.FilesSkipped++
		mu.Unlock()
		return
	}

	if err := os.Rename(srcPath, destPath); err != nil {
		mu.Lock()
		result.Errors = append(result.Errors, fmt.Errorf("move %q → %q: %w", srcPath, destPath, err))
		result.FilesSkipped++
		mu.Unlock()
		return
	}

	mu.Lock()
	journal.Record(srcPath, destPath)
	result.FilesMoved++
	mu.Unlock()

	o.reportProgress(Progress{
		FilesTotal:     total,
		FilesProcessed: processed,
		FilesMoved:     result.FilesMoved,
		FilesSkipped:   result.FilesSkipped,
		CurrentFile:    name,
		Status:         "moving",
	})
}

// PreviewTree returns a hierarchical preview of what organize would do.
func (o *Organizer) PreviewTree(dir string) (*TreePreview, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("preview: failed to read directory %q: %w", dir, err)
	}

	tree := &TreePreview{
		SourceDir:  dir,
		Categories: make(map[string]*CategoryPreview),
	}

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || strings.HasSuffix(strings.ToLower(name), ".lnk") || entry.IsDir() {
			continue
		}

		srcPath := filepath.Join(dir, name)
		if info, err := os.Lstat(srcPath); err == nil && info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		mimeType, _ := detector.Detect(srcPath)
		match := o.engine.Match(name, mimeType)
		category := "Other"
		if match != nil {
			category = match.Category
		}

		if tree.Categories[category] == nil {
			tree.Categories[category] = &CategoryPreview{Name: category}
		}
		tree.Categories[category].Files = append(tree.Categories[category].Files, name)
	}

	return tree, nil
}

// resolveDestination determines the full destination path for a file.
func (o *Organizer) resolveDestination(src, destDir string) string {
	base := filepath.Base(src)
	candidate := filepath.Join(destDir, base)

	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}

	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	for i := 1; ; i++ {
		candidate = filepath.Join(destDir, fmt.Sprintf("%s_%d%s", name, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

// backupJournal creates a timestamped backup of the journal file.
func backupJournal(journalPath string) error {
	if _, err := os.Stat(journalPath); os.IsNotExist(err) {
		return nil
	}

	backupDir := filepath.Join(filepath.Dir(journalPath), "journal-backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(backupDir, fmt.Sprintf("journal-%s.json", timestamp))

	data, err := os.ReadFile(journalPath)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, data, 0o644)
}
