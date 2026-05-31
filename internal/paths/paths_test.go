package paths

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDataDir_ReturnsNonEmpty(t *testing.T) {
	dir := DataDir()
	if dir == "" {
		t.Fatal("DataDir() returned empty string")
	}
}

func TestDataDir_ContainsTidyComponent(t *testing.T) {
	dir := DataDir()
	// Split the path into components and check "tidy" is one of them.
	parts := strings.Split(filepath.ToSlash(dir), "/")
	found := false
	for _, p := range parts {
		if p == "tidy" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("DataDir() = %q does not contain \"tidy\" as a path component", dir)
	}
}

func TestJournalPath_EndsWithJournalJSON(t *testing.T) {
	jp := JournalPath()
	base := filepath.Base(jp)
	if base != "journal.json" {
		t.Fatalf("filepath.Base(JournalPath()) = %q, want \"journal.json\"", base)
	}
}

func TestJournalPath_UnderDataDir(t *testing.T) {
	dir := DataDir()
	jp := JournalPath()
	// JournalPath should be DataDir + separator + "journal.json"
	expected := filepath.Join(dir, "journal.json")
	if jp != expected {
		t.Fatalf("JournalPath() = %q, want %q", jp, expected)
	}
	// Also verify prefix relationship.
	if !strings.HasPrefix(jp, dir) {
		t.Fatalf("JournalPath() = %q is not under DataDir() = %q", jp, dir)
	}
}
