package organizer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/YousefMohiey/tidy/internal/config"
)

func TestJournal_RecordAndFields(t *testing.T) {
	j := NewJournal("/src")
	if j == nil {
		t.Fatal("NewJournal returned nil")
	}
	if j.SourceDir != "/src" {
		t.Errorf("SourceDir = %q, want /src", j.SourceDir)
	}
	if len(j.Operations) != 0 {
		t.Errorf("new journal should have 0 operations, got %d", len(j.Operations))
	}

	j.Record("/src/a.txt", "/dst/Other/a.txt", "Other", 42)
	if len(j.Operations) != 1 {
		t.Fatalf("after Record: len = %d, want 1", len(j.Operations))
	}
	op := j.Operations[0]
	if op.Source != "/src/a.txt" {
		t.Errorf("Source = %q", op.Source)
	}
	if op.Destination != "/dst/Other/a.txt" {
		t.Errorf("Destination = %q", op.Destination)
	}
	if op.Category != "Other" {
		t.Errorf("Category = %q", op.Category)
	}
	if op.Size != 42 {
		t.Errorf("Size = %d", op.Size)
	}
	if op.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
}

func TestJournal_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	savePath := filepath.Join(dir, "subdir", "journal.json")

	j := NewJournal("/some/src")
	j.Record("/some/src/a.jpg", "/some/src/Images/a.jpg", "Images", 100)
	j.Record("/some/src/b.go", "/some/src/Code/b.go", "Code", 200)

	if err := j.Save(savePath); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file was created (including subdir)
	if _, err := os.Stat(savePath); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}

	loaded, err := LoadJournal(savePath)
	if err != nil {
		t.Fatalf("LoadJournal: %v", err)
	}

	if loaded.SourceDir != "/some/src" {
		t.Errorf("loaded SourceDir = %q", loaded.SourceDir)
	}
	if len(loaded.Operations) != 2 {
		t.Fatalf("loaded ops = %d, want 2", len(loaded.Operations))
	}
	if loaded.Operations[0].Category != "Images" {
		t.Errorf("op[0].Category = %q", loaded.Operations[0].Category)
	}
	if loaded.Operations[1].Category != "Code" {
		t.Errorf("op[1].Category = %q", loaded.Operations[1].Category)
	}
	if loaded.Operations[0].Size != 100 {
		t.Errorf("op[0].Size = %d", loaded.Operations[0].Size)
	}
}

func TestJournal_LoadMissing(t *testing.T) {
	_, err := LoadJournal(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil {
		t.Fatal("expected error loading missing file")
	}
}

func TestJournal_LoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(p, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadJournal(p); err == nil {
		t.Fatal("expected error on invalid JSON")
	}
}

// TestJournal_Undo verifies that Undo moves files back to their original locations.
func TestJournal_Undo(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcA := filepath.Join(srcDir, "a.jpg")
	dstA := filepath.Join(dstDir, "a.jpg")
	srcB := filepath.Join(srcDir, "b.go")
	dstB := filepath.Join(dstDir, "b.go")

	// Create source files
	if err := os.WriteFile(srcA, []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcB, []byte("code"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Simulate what organizer does: move src -> dst
	if err := os.Rename(srcA, dstA); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(srcB, dstB); err != nil {
		t.Fatal(err)
	}

	// Build journal recording the moves
	j := NewJournal(srcDir)
	j.Record(srcA, dstA, "Images", 3)
	j.Record(srcB, dstB, "Code", 4)

	// Save and reload to exercise full round-trip
	savePath := filepath.Join(t.TempDir(), "journal.json")
	if err := j.Save(savePath); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := LoadJournal(savePath)
	if err != nil {
		t.Fatalf("LoadJournal: %v", err)
	}

	// Undo
	restored, err := loaded.Undo()
	if err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if restored != 2 {
		t.Errorf("restored = %d, want 2", restored)
	}

	// Verify files are back at source
	data, err := os.ReadFile(srcA)
	if err != nil {
		t.Fatalf("read srcA after undo: %v", err)
	}
	if string(data) != "img" {
		t.Errorf("srcA content = %q", string(data))
	}
	data, err = os.ReadFile(srcB)
	if err != nil {
		t.Fatalf("read srcB after undo: %v", err)
	}
	if string(data) != "code" {
		t.Errorf("srcB content = %q", string(data))
	}

	// Verify destinations are gone
	if _, err := os.Stat(dstA); !os.IsNotExist(err) {
		t.Error("dstA should not exist after undo")
	}
	if _, err := os.Stat(dstB); !os.IsNotExist(err) {
		t.Error("dstB should not exist after undo")
	}
}

// TestJournal_Undo_MissingDest verifies that missing destination files are skipped gracefully.
func TestJournal_Undo_MissingDest(t *testing.T) {
	j := NewJournal("/x")
	j.Record("/x/a.txt", "/x/Other/a.txt", "Other", 1)

	restored, err := j.Undo()
	if err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if restored != 0 {
		t.Errorf("restored = %d, want 0 (dest missing)", restored)
	}
}

// TestJournal_Undo_SourceAlreadyExists verifies that undo skips when source already exists.
func TestJournal_Undo_SourceAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	dst := filepath.Join(dir, "moved", "a.txt")

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("orig"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("moved"), 0o644); err != nil {
		t.Fatal(err)
	}

	j := NewJournal(dir)
	j.Record(src, dst, "Other", 4)

	restored, err := j.Undo()
	if err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if restored != 0 {
		t.Errorf("restored = %d, want 0 (src exists)", restored)
	}

	// src should still have original content (not overwritten)
	data, _ := os.ReadFile(src)
	if string(data) != "orig" {
		t.Errorf("src was overwritten: %q", string(data))
	}
	// dst should still exist
	if _, err := os.Stat(dst); err != nil {
		t.Errorf("dst should still exist: %v", err)
	}
}

// TestJournal_Undo_RecreatesSourceParentDir verifies undo recreates missing parent dirs.
func TestJournal_Undo_RecreatesSourceParentDir(t *testing.T) {
	root := t.TempDir()
	srcSubdir := filepath.Join(root, "src", "nested")
	src := filepath.Join(srcSubdir, "a.txt")
	dst := filepath.Join(root, "dst", "a.txt")

	// Create src, move to dst, then remove the src nested dir entirely.
	if err := os.MkdirAll(srcSubdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(src, dst); err != nil {
		t.Fatal(err)
	}
	// Remove the nested source dir to prove undo recreates it.
	if err := os.RemoveAll(filepath.Join(root, "src")); err != nil {
		t.Fatal(err)
	}

	j := NewJournal(root)
	j.Record(src, dst, "Other", 1)

	restored, err := j.Undo()
	if err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if restored != 1 {
		t.Errorf("restored = %d, want 1", restored)
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("src should exist after undo: %v", err)
	}
}

// TestJournal_OrganizeIntegration exercises Organize -> Save -> Load -> Undo end-to-end.
func TestJournal_OrganizeIntegration(t *testing.T) {
	srcDir := t.TempDir()
	journalDir := t.TempDir()

	// Create files in source
	if err := os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("img"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "b.go"), []byte("code"), 0o644); err != nil {
		t.Fatal(err)
	}

	o := New(config.Default(), Options{JournalDir: journalDir})
	result, err := o.Organize(srcDir)
	if err != nil {
		t.Fatalf("Organize: %v", err)
	}
	if result.FilesMoved != 2 {
		t.Fatalf("FilesMoved = %d, want 2", result.FilesMoved)
	}

	// Journal should have been saved
	journalPath := filepath.Join(journalDir, "journal.json")
	loaded, err := LoadJournal(journalPath)
	if err != nil {
		t.Fatalf("LoadJournal: %v", err)
	}
	if len(loaded.Operations) != 2 {
		t.Fatalf("operations = %d, want 2", len(loaded.Operations))
	}

	// Undo should restore
	restored, err := loaded.Undo()
	if err != nil {
		t.Fatalf("Undo: %v", err)
	}
	if restored != 2 {
		t.Errorf("restored = %d, want 2", restored)
	}

	// Source files back
	if _, err := os.Stat(filepath.Join(srcDir, "a.jpg")); err != nil {
		t.Errorf("a.jpg should be back: %v", err)
	}
	if _, err := os.Stat(filepath.Join(srcDir, "b.go")); err != nil {
		t.Errorf("b.go should be back: %v", err)
	}
}
