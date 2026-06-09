// Package watcher provides real-time file system monitoring that detects new files
// in a directory and automatically organizes them using the organizer package.
package watcher

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/YousefMohiey/tidy/internal/config"
	"github.com/YousefMohiey/tidy/internal/organizer"
)

// debounceDelay is the quiet period before a file is considered fully written.
const debounceDelay = 500 * time.Millisecond

// tickInterval is how often we check for settled files in the debounce map.
const tickInterval = 200 * time.Millisecond

// Watcher monitors a directory for new files and auto-organizes them.
type Watcher struct {
	dir       string
	organizer *organizer.Organizer
	Output    io.Writer
}

func (w *Watcher) output() io.Writer {
	if w.Output != nil {
		return w.Output
	}
	return os.Stdout
}

// New creates a Watcher that monitors dir and organizes files using the provided
// config and organizer options.
func New(dir string, cfg *config.Config, opts organizer.Options) *Watcher {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = filepath.Clean(dir)
	}

	return &Watcher{
		dir:       absDir,
		organizer: organizer.New(cfg, opts),
	}
}

// Watch starts monitoring the directory for new files.
// Blocks until context is cancelled or an unrecoverable error occurs.
// On new file creation, waits briefly for the file to be fully written (debounced),
// then organizes it using the organizer.
func (w *Watcher) Watch(ctx context.Context) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watcher: failed to create fsnotify watcher: %w", err)
	}
	defer fsw.Close()

	if err := fsw.Add(w.dir); err != nil {
		return fmt.Errorf("watcher: failed to watch directory %q: %w", w.dir, err)
	}

	fmt.Fprintf(w.output(), "watching: %s\n", w.dir)

	// pending tracks files being written: path -> last event time.
	pending := make(map[string]time.Time)
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(w.output(), "watcher: shutting down")
			return ctx.Err()

		case event, ok := <-fsw.Events:
			if !ok {
				return nil
			}
			w.handleEvent(event, pending)

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(w.output(), "watcher error: %v\n", err)

		case now := <-ticker.C:
			w.flushReady(now, pending)
		}
	}
}

// handleEvent processes a single fsnotify event, adding qualifying files to the
// pending debounce map.
func (w *Watcher) handleEvent(event fsnotify.Event, pending map[string]time.Time) {
	// React only to Create and Write events.
	if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) {
		return
	}

	// Only process files in the top-level watched directory.
	eventDir := filepath.Clean(filepath.Dir(event.Name))
	if eventDir != w.dir {
		return
	}

	// Skip hidden files (name starts with '.').
	name := filepath.Base(event.Name)
	if strings.HasPrefix(name, ".") {
		return
	}

	// Skip directories — only process regular files.
	info, err := os.Stat(event.Name)
	if err != nil {
		// File may have been removed between event and stat; skip silently.
		return
	}
	if info.IsDir() {
		return
	}

	pending[event.Name] = time.Now()
}

// flushReady checks the pending map for files that have been idle longer than
// the debounce delay, removes them, and triggers a single organize pass.
func (w *Watcher) flushReady(now time.Time, pending map[string]time.Time) {
	var readyCount int
	for path, last := range pending {
		if now.Sub(last) >= debounceDelay {
			delete(pending, path)
			readyCount++
		}
	}

	if readyCount == 0 {
		return
	}

	go w.runOrganize()
}

// runOrganize calls the organizer on the watched directory and logs results.
func (w *Watcher) runOrganize() {
	result, err := w.organizer.Organize(w.dir)
	if err != nil {
		fmt.Fprintf(w.output(), "organize error: %v\n", err)
		return
	}

	// Log per-file errors from the result.
	for _, e := range result.Errors {
		fmt.Fprintf(w.output(), "organize file error: %v\n", e)
	}

	// Log each successful move.
	for _, m := range result.Moves {
		from := filepath.Base(m.Source)
		to, relErr := filepath.Rel(w.dir, m.Destination)
		if relErr != nil {
			to = m.Destination
		}
		fmt.Fprintf(w.output(), "organized: %s → %s\n", from, to)
	}

	if result.FilesProcessed > 0 {
		fmt.Fprintf(w.output(), "processed: %d, moved: %d, skipped: %d\n",
			result.FilesProcessed, result.FilesMoved, result.FilesSkipped)
	}
}
