package tui

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/YousefMohiey/tidy/internal/organizer"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type organizeResultMsg struct {
	result *organizer.Result
	err    error
	dryRun bool
}

type organizeProgressMsg struct {
	progress organizer.Progress
}

type categoryStat struct {
	Count int
	Size  int64
}

func (m model) runOrganize(dryRun bool) (tea.Model, tea.Cmd) {
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

	if dryRun {
		m.status = "Previewing..."
	} else {
		m.status = "Organizing..."
	}
	m.statusStyle = accentStyle

	cfg := m.data.Config
	dir := m.data.SourceDir
	return m, func() tea.Msg {
		opts := organizer.Options{
			DryRun: dryRun,
			OnProgress: func(p organizer.Progress) {
				// Bubble Tea doesn't support sending msgs from callbacks easily,
				// so we just store progress in the result.
			},
		}
		org := organizer.New(cfg, opts)
		result, err := org.Organize(dir)
		return organizeResultMsg{result: result, err: err, dryRun: dryRun}
	}
}

func (m model) resultsLines(width int) []string {
	var lines []string

	if m.lastResult == nil {
		lines = append(lines,
			"",
			"  "+mutedStyle.Render("No actions performed yet."),
			"  "+mutedStyle.Render("Use the Home tab to run operations."),
		)
		return lines
	}

	r := m.lastResult

	actionLabel := secondaryStyle.Render(r.Action)
	timeLabel := mutedStyle.Render(r.Timestamp.Format("2006-01-02 15:04:05"))
	lines = append(lines,
		"  "+labelStyle.Render("Last action: ")+actionLabel+" "+timeLabel,
	)

	if r.Success {
		lines = append(lines,
			"  "+labelStyle.Render("Result: ")+successStyle.Render(r.Summary),
		)
	} else {
		lines = append(lines,
			"  "+labelStyle.Render("Result: ")+errorStyle.Render(r.Summary),
		)
	}

	lines = append(lines, "")

	if len(r.Details) == 0 {
		lines = append(lines, "  "+mutedStyle.Render("No details available."))
		return lines
	}

	lines = append(lines, "  "+secondaryStyle.Render("Details:"))

	for _, d := range r.Details {
		line := "    " + d
		avail := width - 4
		if avail < 10 {
			avail = 10
		}
		if lipgloss.Width(line) > avail {
			line = truncateMiddle(line, avail)
		}
		lines = append(lines, line)
	}

	if len(r.Moves) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  "+secondaryStyle.Render("Category Breakdown:"))
		cats := categoryBreakdown(r.Moves)
		catNames := make([]string, 0, len(cats))
		for name := range cats {
			catNames = append(catNames, name)
		}
		sort.Strings(catNames)
		for _, name := range catNames {
			st := cats[name]
			line := fmt.Sprintf("    📁 %s: %d files (%s)", name, st.Count, formatSize(st.Size))
			lines = append(lines, line)
		}
	}

	return lines
}

func actionLabel(dryRun bool) string {
	if dryRun {
		return "Preview"
	}
	return "Organize"
}

func formatMoves(moves []organizer.MoveRecord, sourceDir string) []string {
	if len(moves) == 0 {
		return nil
	}

	maxSrcLen := 0
	type moveEntry struct {
		srcName string
		dstName string
	}
	entries := make([]moveEntry, 0, len(moves))

	for _, mv := range moves {
		src := filepath.Base(mv.Source)
		dst := mv.Destination
		if sourceDir != "" {
			if rel, err := filepath.Rel(sourceDir, dst); err == nil {
				dst = rel
			}
		}
		if len(src) > maxSrcLen {
			maxSrcLen = len(src)
		}
		entries = append(entries, moveEntry{srcName: src, dstName: dst})
	}

	if maxSrcLen > 30 {
		maxSrcLen = 30
	}

	result := make([]string, 0, len(entries))
	for _, e := range entries {
		srcName := e.srcName
		if len(srcName) > 30 {
			srcName = truncateMiddle(srcName, 30)
		}
		line := fmt.Sprintf("%-*s", maxSrcLen, srcName) +
			" " + arrowStyle.Render("→") + " " + valueStyle.Render(e.dstName)
		result = append(result, line)
	}

	return result
}

func categoryBreakdown(moves []organizer.MoveRecord) map[string]categoryStat {
	cats := make(map[string]categoryStat)
	for _, mv := range moves {
		cat := mv.Category
		if cat == "" {
			cat = "Other"
		}
		st := cats[cat]
		st.Count++
		st.Size += mv.Size
		cats[cat] = st
	}
	return cats
}

func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
