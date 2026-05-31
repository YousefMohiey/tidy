package organizer

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/YousefMohiey/tidy/internal/config"
)

// createFile writes a small file with the given content (or empty if content is nil).
func createFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if content == nil {
		content = []byte("x")
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func fileExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	t.Fatalf("stat %s: %v", path, err)
	return false
}

func TestOrganize_Basic(t *testing.T) {
	dir := t.TempDir()

	// .jpg -> Images, .go -> Code (both in default config)
	createFile(t, filepath.Join(dir, "a.jpg"), nil)
	createFile(t, filepath.Join(dir, "b.go"), nil)

	o := New(config.Default(), Options{JournalDir: t.TempDir()})
	result, err := o.Organize(dir)
	if err != nil {
		t.Fatalf("Organize: %v", err)
	}

	if result.FilesMoved != 2 {
		t.Errorf("FilesMoved = %d, want 2", result.FilesMoved)
	}
	if result.FilesSkipped != 0 {
		t.Errorf("FilesSkipped = %d, want 0", result.FilesSkipped)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want none", result.Errors)
	}

	// Verify files moved to correct category dirs
	if !fileExists(t, filepath.Join(dir, "Images", "a.jpg")) {
		t.Error("expected Images/a.jpg to exist")
	}
	if !fileExists(t, filepath.Join(dir, "Code", "b.go")) {
		t.Error("expected Code/b.go to exist")
	}

	// Verify source files are gone
	if fileExists(t, filepath.Join(dir, "a.jpg")) {
		t.Error("source a.jpg should not exist")
	}
	if fileExists(t, filepath.Join(dir, "b.go")) {
		t.Error("source b.go should not exist")
	}
}

func TestOrganize_ConflictResolution(t *testing.T) {
	dir := t.TempDir()

	// Create source file
	createFile(t, filepath.Join(dir, "photo.jpg"), []byte("new"))

	// Pre-create destination with same filename -> conflict
	createFile(t, filepath.Join(dir, "Images", "photo.jpg"), []byte("existing"))

	o := New(config.Default(), Options{JournalDir: t.TempDir()})
	result, err := o.Organize(dir)
	if err != nil {
		t.Fatalf("Organize: %v", err)
	}

	if result.FilesMoved != 1 {
		t.Errorf("FilesMoved = %d, want 1", result.FilesMoved)
	}

	// Original in Images should be untouched
	data, err := os.ReadFile(filepath.Join(dir, "Images", "photo.jpg"))
	if err != nil {
		t.Fatalf("read existing: %v", err)
	}
	if string(data) != "existing" {
		t.Errorf("existing file was overwritten: got %q", string(data))
	}

	// New file should be photo_1.jpg
	if !fileExists(t, filepath.Join(dir, "Images", "photo_1.jpg")) {
		t.Error("expected Images/photo_1.jpg to exist")
	}
}

func TestOrganize_DryRun(t *testing.T) {
	dir := t.TempDir()

	createFile(t, filepath.Join(dir, "a.jpg"), nil)
	createFile(t, filepath.Join(dir, "b.go"), nil)

	o := New(config.Default(), Options{DryRun: true, JournalDir: t.TempDir()})
	result, err := o.Organize(dir)
	if err != nil {
		t.Fatalf("Organize: %v", err)
	}

	if !result.DryRun {
		t.Error("result.DryRun should be true")
	}
	if result.FilesMoved != 2 {
		t.Errorf("FilesMoved = %d, want 2 (dry-run still counts)", result.FilesMoved)
	}

	// Files must still be in place
	if !fileExists(t, filepath.Join(dir, "a.jpg")) {
		t.Error("source a.jpg should still exist in dry-run")
	}
	if !fileExists(t, filepath.Join(dir, "b.go")) {
		t.Error("source b.go should still exist in dry-run")
	}

	// Category dirs should NOT have been created
	if fileExists(t, filepath.Join(dir, "Images")) {
		t.Error("Images/ should not exist in dry-run")
	}
	if fileExists(t, filepath.Join(dir, "Code")) {
		t.Error("Code/ should not exist in dry-run")
	}

	// Moves should be populated with what would happen
	if len(result.Moves) != 2 {
		t.Fatalf("len(Moves) = %d, want 2", len(result.Moves))
	}
}

func TestOrganize_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	o := New(config.Default(), Options{JournalDir: t.TempDir()})
	result, err := o.Organize(dir)
	if err != nil {
		t.Fatalf("Organize: %v", err)
	}

	if result.FilesMoved != 0 {
		t.Errorf("FilesMoved = %d, want 0", result.FilesMoved)
	}
	if result.FilesProcessed != 0 {
		t.Errorf("FilesProcessed = %d, want 0", result.FilesProcessed)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want none", result.Errors)
	}
}

func TestOrganize_SkipsHiddenAndDirs(t *testing.T) {
	dir := t.TempDir()

	createFile(t, filepath.Join(dir, "a.jpg"), nil)
	createFile(t, filepath.Join(dir, ".hidden.jpg"), nil)       // dotfile - skip
	if err := os.MkdirAll(filepath.Join(dir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	o := New(config.Default(), Options{JournalDir: t.TempDir()})
	result, err := o.Organize(dir)
	if err != nil {
		t.Fatalf("Organize: %v", err)
	}

	if result.FilesMoved != 1 {
		t.Errorf("FilesMoved = %d, want 1", result.FilesMoved)
	}
	if !fileExists(t, filepath.Join(dir, ".hidden.jpg")) {
		t.Error(".hidden.jpg should not have been moved")
	}
	if !fileExists(t, filepath.Join(dir, "subdir")) {
		t.Error("subdir should still exist")
	}
}

func TestPreviewTree(t *testing.T) {
	dir := t.TempDir()

	createFile(t, filepath.Join(dir, "a.jpg"), nil)
	createFile(t, filepath.Join(dir, "b.png"), nil)
	createFile(t, filepath.Join(dir, "c.go"), nil)
	createFile(t, filepath.Join(dir, "unknown.xyz"), nil)

	o := New(config.Default(), Options{JournalDir: t.TempDir()})
	tree, err := o.PreviewTree(dir)
	if err != nil {
		t.Fatalf("PreviewTree: %v", err)
	}

	if tree.SourceDir != dir {
		t.Errorf("SourceDir = %q, want %q", tree.SourceDir, dir)
	}

	// Check Images category
	img := tree.Categories["Images"]
	if img == nil {
		t.Fatal("expected Images category")
	}
	sort.Strings(img.Files)
	if len(img.Files) != 2 || img.Files[0] != "a.jpg" || img.Files[1] != "b.png" {
		t.Errorf("Images files = %v, want [a.jpg b.png]", img.Files)
	}

	// Check Code category
	code := tree.Categories["Code"]
	if code == nil {
		t.Fatal("expected Code category")
	}
	if len(code.Files) != 1 || code.Files[0] != "c.go" {
		t.Errorf("Code files = %v, want [c.go]", code.Files)
	}

	// Unknown -> Other
	other := tree.Categories["Other"]
	if other == nil {
		t.Fatal("expected Other category for unknown.xyz")
	}
	if len(other.Files) != 1 || other.Files[0] != "unknown.xyz" {
		t.Errorf("Other files = %v, want [unknown.xyz]", other.Files)
	}

	// Verify nothing was moved (preview only)
	if !fileExists(t, filepath.Join(dir, "a.jpg")) {
		t.Error("a.jpg should still exist after preview")
	}
}
