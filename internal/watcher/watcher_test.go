package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/YousefMohiey/tidy/internal/config"
	"github.com/YousefMohiey/tidy/internal/organizer"
)

func TestNew(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	opts := organizer.Options{JournalDir: t.TempDir()}

	w := New(dir, cfg, opts)

	if w == nil {
		t.Fatal("New() returned nil")
	}

	absDir, _ := filepath.Abs(dir)
	if w.dir != absDir {
		t.Errorf("w.dir = %q, want %q", w.dir, absDir)
	}

	if w.organizer == nil {
		t.Error("w.organizer is nil")
	}
}

func TestWatch_OrganizesFile(t *testing.T) {
	watchDir := t.TempDir()
	journalDir := t.TempDir()
	cfg := config.Default()
	opts := organizer.Options{JournalDir: journalDir}

	w := New(watchDir, cfg, opts)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Watch(ctx)
	}()

	// Give fsnotify time to initialize and start monitoring.
	time.Sleep(500 * time.Millisecond)

	// Drop a .jpg file into the watched directory.
	testFile := filepath.Join(watchDir, "photo.jpg")
	if err := os.WriteFile(testFile, []byte("fake jpg data"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Poll until the file appears in Images/ or we hit the deadline.
	// debounce (500ms) + tick (200ms) + organize + generous Windows margin.
	movedFile := filepath.Join(watchDir, "Images", "photo.jpg")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(movedFile); err == nil {
			cancel()
			<-errCh
			return
		}
		time.Sleep(200 * time.Millisecond)
	}

	cancel()
	<-errCh
	t.Fatal("timed out waiting for photo.jpg to be organized into Images/")
}

func TestWatch_ContextCancellation(t *testing.T) {
	watchDir := t.TempDir()
	journalDir := t.TempDir()
	cfg := config.Default()
	opts := organizer.Options{JournalDir: journalDir}

	w := New(watchDir, cfg, opts)

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Watch(ctx)
	}()

	// Let the watcher start up.
	time.Sleep(200 * time.Millisecond)

	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Errorf("Watch() returned %v, want context.Canceled", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Watch() did not return within 3s after context cancellation")
	}
}
