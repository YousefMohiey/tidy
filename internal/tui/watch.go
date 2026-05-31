package tui

import (
	"context"
	"io"

	"github.com/YousefMohiey/tidy/internal/organizer"
	"github.com/YousefMohiey/tidy/internal/watcher"

	tea "github.com/charmbracelet/bubbletea"
)

type watchDoneMsg struct{}

func (m model) toggleWatch() (tea.Model, tea.Cmd) {
	if m.watching {
		if m.watchCancel != nil {
			m.watchCancel()
		}
		return m, nil
	}

	if m.data.Config == nil {
		m.status = "Error: no configuration loaded"
		m.statusStyle = errorStyle
		return m, nil
	}
	if m.data.SourceDir == "" {
		m.status = "Error: no source directory set (press e to set)"
		m.statusStyle = errorStyle
		return m, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.watching = true
	m.watchCancel = cancel
	m.status = "Watching... (press w to stop)"
	m.statusStyle = successStyle

	cfg := m.data.Config
	dir := m.data.SourceDir
	return m, func() tea.Msg {
		w := watcher.New(dir, cfg, organizer.Options{})
		w.Output = io.Discard
		_ = w.Watch(ctx)
		return watchDoneMsg{}
	}
}
