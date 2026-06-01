package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/YousefMohiey/tidy/internal/organizer"
	"github.com/YousefMohiey/tidy/internal/paths"

	tea "github.com/charmbracelet/bubbletea"
)

type undoResultMsg struct {
	restored int
	err      error
}

type UndoEntry struct {
	Timestamp   time.Time
	FilesMoved  int
	SourceDir   string
	JournalPath string
}

func (m model) handleDetailsKey(key string) (tea.Model, tea.Cmd) {
	if len(m.undoHistory) == 0 {
		return m.handleScrollKey(key)
	}
	switch key {
	case "j", "down":
		if m.undoSelected < len(m.undoHistory)-1 {
			m.undoSelected++
		}
		return m, nil
	case "k", "up":
		if m.undoSelected > 0 {
			m.undoSelected--
		}
		return m, nil
	case "enter":
		if m.undoSelected < len(m.undoHistory) {
			entry := m.undoHistory[m.undoSelected]
			m.confirming = true
			m.confirmMsg = fmt.Sprintf("Undo %d files from %s?", entry.FilesMoved, filepath.Base(entry.SourceDir))
			m.confirmAction = "undo-entry"
		}
		return m, nil
	}
	return m.handleScrollKey(key)
}

func (m model) runUndoEntry(idx int) (tea.Model, tea.Cmd) {
	if idx >= len(m.undoHistory) {
		return m, nil
	}
	entry := m.undoHistory[idx]
	m.status = "Undoing..."
	m.statusStyle = accentStyle

	return m, func() tea.Msg {
		journal, err := organizer.LoadJournal(entry.JournalPath)
		if err != nil {
			return undoResultMsg{err: err}
		}
		restored, err := journal.Undo()
		if err == nil {
			_ = os.Remove(entry.JournalPath)
		}
		return undoResultMsg{restored: restored, err: err}
	}
}

func (m model) requestUndo() (tea.Model, tea.Cmd) {
	if m.data.Journal == nil {
		m.status = "No operations to undo"
		m.statusStyle = errorStyle
		return m, nil
	}
	m.confirming = true
	m.confirmMsg = fmt.Sprintf("Undo %d files from %s?", len(m.data.Journal.Operations), filepath.Base(m.data.Journal.SourceDir))
	m.confirmAction = "undo"
	return m, nil
}

func (m model) runUndoLast() (tea.Model, tea.Cmd) {
	m.status = "Undoing last operation..."
	m.statusStyle = accentStyle

	journalPath := paths.JournalPath()
	return m, func() tea.Msg {
		journal, err := organizer.LoadJournal(journalPath)
		if err != nil {
			return undoResultMsg{err: err}
		}
		restored, err := journal.Undo()
		if err == nil {
			_ = os.Remove(journalPath)
		}
		return undoResultMsg{restored: restored, err: err}
	}
}

func (m *model) reloadJournal() {
	jPath := paths.JournalPath()
	if jPath == "" {
		m.data.Journal = nil
		return
	}
	journal, err := organizer.LoadJournal(jPath)
	if err != nil {
		m.data.Journal = nil
		return
	}
	m.data.Journal = journal
}

func (m model) detailsLines(width int) []string {
	var lines []string

	if len(m.undoHistory) == 0 {
		lines = append(lines,
			"",
			"  "+mutedStyle.Render("No operations recorded yet."),
			"  "+mutedStyle.Render("Operations will appear here after organizing files."),
		)
		return lines
	}

	lines = append(lines, "")
	lines = append(lines, "  "+titleStyle.Render("Recent Operations"))
	lines = append(lines, "")

	for i, entry := range m.undoHistory {
		indicator := "  "
		timeStr := relativeTime(entry.Timestamp)
		summary := fmt.Sprintf("%s — %d files from %s", timeStr, entry.FilesMoved, truncateMiddle(entry.SourceDir, 40))

		if i == m.undoSelected {
			indicator = accentStyle.Render("> ")
			lines = append(lines, indicator+accentStyle.Render(summary))

			if journal, err := organizer.LoadJournal(entry.JournalPath); err == nil && len(journal.Operations) > 0 {
				cats := make(map[string]int)
				for _, op := range journal.Operations {
					cat := op.Category
					if cat == "" {
						cat = "Other"
					}
					cats[cat]++
				}
				var parts []string
				for cat, count := range cats {
					parts = append(parts, fmt.Sprintf("%s: %d", cat, count))
				}
				sort.Strings(parts)
				catLine := "      " + mutedStyle.Render(strings.Join(parts, "  "))
				lines = append(lines, catLine)
			}

			undoHint := "      " + mutedStyle.Render("[enter] undo this operation")
			lines = append(lines, undoHint)
		} else {
			lines = append(lines, indicator+valueStyle.Render(summary))
		}
	}

	return lines
}

func relativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "Just now"
	}
	if diff < time.Hour {
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "Today " + t.Format("15:04")
		}
		return "Today " + t.Format("15:04")
	}
	if diff < 48*time.Hour {
		return "Yesterday " + t.Format("15:04")
	}
	if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	}
	return t.Format("2006-01-02 15:04")
}

func loadUndoHistory() []UndoEntry {
	dataDir := paths.DataDir()
	if dataDir == "" {
		return nil
	}

	var entries []UndoEntry

	journalPath := paths.JournalPath()
	if journal, err := organizer.LoadJournal(journalPath); err == nil && len(journal.Operations) > 0 {
		entries = append(entries, UndoEntry{
			Timestamp:   journal.Timestamp,
			FilesMoved:  len(journal.Operations),
			SourceDir:   journal.SourceDir,
			JournalPath: journalPath,
		})
	}

	backupDir := filepath.Join(dataDir, "journal-backups")
	if backupEntries, err := os.ReadDir(backupDir); err == nil {
		for _, entry := range backupEntries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			backupPath := filepath.Join(backupDir, entry.Name())
			if journal, err := organizer.LoadJournal(backupPath); err == nil && len(journal.Operations) > 0 {
				entries = append(entries, UndoEntry{
					Timestamp:   journal.Timestamp,
					FilesMoved:  len(journal.Operations),
					SourceDir:   journal.SourceDir,
					JournalPath: backupPath,
				})
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	if len(entries) > 5 {
		entries = entries[:5]
	}

	return entries
}
