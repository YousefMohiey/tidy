package dedup

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestScanner(t *testing.T) *Scanner {
	t.Helper()
	return &Scanner{
		MinSize:  1,
		MaxSize:  1073741824,
		MaxDepth: 0,
		Cache: &HashCache{
			entries: make(map[string]*CacheEntry),
			path:    filepath.Join(t.TempDir(), "cache.json"),
		},
	}
}

func TestScanDuplicateFiles(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "a.txt"), "hello world")
	writeFile(t, filepath.Join(dir, "b.txt"), "hello world")
	writeFile(t, filepath.Join(dir, "c.txt"), "different")

	s := newTestScanner(t)
	result, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if result.TotalFiles != 3 {
		t.Errorf("TotalFiles = %d, want 3", result.TotalFiles)
	}

	if len(result.DuplicateGroups) != 1 {
		t.Fatalf("DuplicateGroups count = %d, want 1", len(result.DuplicateGroups))
	}

	g := result.DuplicateGroups[0]
	if len(g.Files) != 2 {
		t.Errorf("group Files count = %d, want 2", len(g.Files))
	}
	if g.Size != 11 {
		t.Errorf("group Size = %d, want 11", g.Size)
	}
	if g.Hash == "" {
		t.Error("group Hash is empty")
	}

	if result.UniqueFiles != 1 {
		t.Errorf("UniqueFiles = %d, want 1", result.UniqueFiles)
	}
}

func TestScanAllUnique(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "x.txt"), "aaa")
	writeFile(t, filepath.Join(dir, "y.txt"), "bbb")
	writeFile(t, filepath.Join(dir, "z.txt"), "ccc")

	s := newTestScanner(t)
	result, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if len(result.DuplicateGroups) != 0 {
		t.Errorf("DuplicateGroups count = %d, want 0", len(result.DuplicateGroups))
	}
	if result.TotalFiles != 3 {
		t.Errorf("TotalFiles = %d, want 3", result.TotalFiles)
	}
	if result.UniqueFiles != 3 {
		t.Errorf("UniqueFiles = %d, want 3", result.UniqueFiles)
	}
}

func TestScanEmptyDir(t *testing.T) {
	dir := t.TempDir()

	s := newTestScanner(t)
	result, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if result.TotalFiles != 0 {
		t.Errorf("TotalFiles = %d, want 0", result.TotalFiles)
	}
	if result.UniqueFiles != 0 {
		t.Errorf("UniqueFiles = %d, want 0", result.UniqueFiles)
	}
	if len(result.DuplicateGroups) != 0 {
		t.Errorf("DuplicateGroups = %d, want 0", len(result.DuplicateGroups))
	}
	if result.WastedBytes != 0 {
		t.Errorf("WastedBytes = %d, want 0", result.WastedBytes)
	}
}

func TestScanNoDirs(t *testing.T) {
	s := newTestScanner(t)
	_, err := s.Scan()
	if err == nil {
		t.Fatal("expected error for no directories, got nil")
	}
}

func TestScanInvalidDir(t *testing.T) {
	s := newTestScanner(t)
	_, err := s.Scan(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatal("expected error for invalid directory, got nil")
	}
}

func TestWastedBytesTotal(t *testing.T) {
	r := &ScanResult{
		DuplicateGroups: []DuplicateGroup{
			{Hash: "a", Size: 100, Files: []string{"f1", "f2", "f3"}},
			{Hash: "b", Size: 50, Files: []string{"f4", "f5"}},
		},
	}

	got := r.WastedBytesTotal()
	// Group a: (3-1)*100 = 200, Group b: (2-1)*50 = 50, total = 250
	want := int64(250)
	if got != want {
		t.Errorf("WastedBytesTotal() = %d, want %d", got, want)
	}
}

func TestWastedBytesTotalEmpty(t *testing.T) {
	r := &ScanResult{}
	if got := r.WastedBytesTotal(); got != 0 {
		t.Errorf("WastedBytesTotal() = %d, want 0", got)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{10240, "10 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{-1, "0 B"},
	}

	for _, tt := range tests {
		got := FormatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestScanWastedBytesInResult(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "dup1.txt"), "duplicate content!")
	writeFile(t, filepath.Join(dir, "dup2.txt"), "duplicate content!")

	s := newTestScanner(t)
	result, err := s.Scan(dir)
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	if len(result.DuplicateGroups) != 1 {
		t.Fatalf("DuplicateGroups = %d, want 1", len(result.DuplicateGroups))
	}

	// 2 files of 18 bytes each: wasted = (2-1)*18 = 18
	if result.WastedBytes != 18 {
		t.Errorf("WastedBytes = %d, want 18", result.WastedBytes)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
