package organizer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MoveRecord represents a single file move operation.
type MoveRecord struct {
	Source      string    `json:"source"`
	Destination string    `json:"destination"`
	Timestamp   time.Time `json:"timestamp"`
}

// Journal tracks a batch of move operations for undo capability.
type Journal struct {
	Operations []MoveRecord `json:"operations"`
	Timestamp  time.Time    `json:"timestamp"` // when the batch started
	SourceDir  string       `json:"source_dir"`
}

// NewJournal creates a new empty journal for a source directory.
func NewJournal(sourceDir string) *Journal {
	return &Journal{
		Operations: []MoveRecord{},
		Timestamp:  time.Now(),
		SourceDir:  sourceDir,
	}
}

// Record adds a move record to the journal.
func (j *Journal) Record(src, dst string) {
	j.Operations = append(j.Operations, MoveRecord{
		Source:      src,
		Destination: dst,
		Timestamp:   time.Now(),
	})
}

// Save writes the journal to a JSON file at the given path.
func (j *Journal) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("journal: failed to create directory %q: %w", dir, err)
	}

	data, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return fmt.Errorf("journal: failed to marshal operations: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("journal: failed to write file %q: %w", path, err)
	}

	return nil
}

// Append loads an existing journal and appends the current operations to it,
// then saves. If no journal exists, falls back to Save.
func (j *Journal) Append(path string) error {
	existing, err := LoadJournal(path)
	if err == nil && existing != nil && existing.SourceDir == j.SourceDir {
		existing.Operations = append(existing.Operations, j.Operations...)
		return existing.Save(path)
	}
	return j.Save(path)
}

// LoadJournal reads a journal from a JSON file.
func LoadJournal(path string) (*Journal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("journal: failed to read file %q: %w", path, err)
	}

	var j Journal
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, fmt.Errorf("journal: failed to parse file %q: %w", path, err)
	}

	return &j, nil
}

// Undo reverses all operations in the journal (moves files back).
// Iterates in reverse order. Missing files are skipped gracefully.
// Returns the number of files successfully restored and any fatal error.
func (j *Journal) Undo() (int, error) {
	restored := 0
	var errs []error

	for i := len(j.Operations) - 1; i >= 0; i-- {
		op := j.Operations[i]

		if _, err := os.Lstat(op.Destination); err != nil {
			if os.IsNotExist(err) {
				// File already gone — skip silently.
				continue
			}
			errs = append(errs, fmt.Errorf("undo: stat %q: %w", op.Destination, err))
			continue
		}

		// Ensure the source directory exists.
		srcDir := filepath.Dir(op.Source)
		if err := os.MkdirAll(srcDir, 0o755); err != nil {
			errs = append(errs, fmt.Errorf("undo: mkdir %q: %w", srcDir, err))
			continue
		}

		if err := os.Rename(op.Destination, op.Source); err != nil {
			errs = append(errs, fmt.Errorf("undo: move %q → %q: %w", op.Destination, op.Source, err))
			continue
		}

		restored++
	}

	if len(errs) > 0 {
		return restored, fmt.Errorf("undo: %d operation(s) failed: %w", len(errs), errs[0])
	}

	return restored, nil
}
