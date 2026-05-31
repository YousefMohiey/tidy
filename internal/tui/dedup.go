package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/YousefMohiey/tidy/internal/dedup"

	tea "github.com/charmbracelet/bubbletea"
)

type dedupResultMsg struct {
	result *dedup.ScanResult
	err    error
}

type dedupProgressMsg struct {
	progress dedup.Progress
}

func (m model) handleDupKey(key string) (tea.Model, tea.Cmd) {
	groups := m.data.DedupScan.DuplicateGroups
	switch key {
	case "esc":
		m.dupMode = ""
		m.scrollY = 0
		return m, nil
	case "up", "k":
		if m.dupMode == "select" {
			if m.dupSelectedGrp > 0 {
				m.dupSelectedGrp--
			}
		} else if m.dupMode == "file" {
			if m.dupSelectedFile > 0 {
				m.dupSelectedFile--
			}
		}
		return m, nil
	case "down", "j":
		if m.dupMode == "select" {
			if m.dupSelectedGrp < len(groups)-1 {
				m.dupSelectedGrp++
			}
		} else if m.dupMode == "file" {
			if m.dupSelectedFile < len(groups[m.dupSelectedGrp].Files)-1 {
				m.dupSelectedFile++
			}
		}
		return m, nil
	case "enter":
		if m.dupMode == "select" {
			m.dupMode = "file"
			m.dupSelectedFile = 0
			return m, nil
		}
		if m.dupMode == "file" {
			return m.runDupKeep()
		}
	case "D":
		if m.dupMode == "select" {
			return m.runDupDelete()
		}
	}
	return m, nil
}

func (m model) runDupDelete() (tea.Model, tea.Cmd) {
	if m.data.DedupScan == nil {
		return m, nil
	}
	idx := m.dupSelectedGrp
	m.status = "Deleting duplicates..."
	m.statusStyle = accentStyle
	scan := m.data.DedupScan
	return m, func() tea.Msg {
		freed, err := scan.DeleteGroup(idx)
		if err != nil {
			return dedupResultMsg{result: scan, err: err}
		}
		return dedupResultMsg{result: scan, err: fmt.Errorf("freed %s", dedup.FormatSize(freed))}
	}
}

func (m model) runDupKeep() (tea.Model, tea.Cmd) {
	if m.data.DedupScan == nil {
		return m, nil
	}
	grpIdx := m.dupSelectedGrp
	fileIdx := m.dupSelectedFile
	m.status = "Removing duplicates..."
	m.statusStyle = accentStyle
	scan := m.data.DedupScan
	return m, func() tea.Msg {
		freed, err := scan.KeepOne(grpIdx, fileIdx)
		if err != nil {
			return dedupResultMsg{result: scan, err: err}
		}
		return dedupResultMsg{result: scan, err: fmt.Errorf("freed %s", dedup.FormatSize(freed))}
	}
}

func (m model) runDedupScan() (tea.Model, tea.Cmd) {
	if m.data.SourceDir == "" {
		m.status = "Error: no source directory set (press e to set)"
		m.statusStyle = errorStyle
		return m, nil
	}

	m.status = "Scanning for duplicates..."
	m.statusStyle = accentStyle

	dir := m.data.SourceDir
	return m, func() tea.Msg {
		scanner := dedup.NewScanner()
		scanner.OnProgress = func(p dedup.Progress) {
			// Progress stored in result for display
		}
		result, err := scanner.Scan(dir)
		return dedupResultMsg{result: result, err: err}
	}
}

func (m model) duplicatesLines(width int) []string {
	var lines []string

	if m.dupMode == "select" || m.dupMode == "file" {
		return m.duplicateResolverLines(width)
	}

	lines = append(lines,
		"  "+accentStyle.Render("[d]")+valueStyle.Render(" Scan for duplicates in current directory"),
		"",
	)

	if m.data.DedupScan == nil {
		lines = append(lines, "  "+mutedStyle.Render("No scan performed yet. Press 'd' to scan."))
		return lines
	}

	r := m.data.DedupScan

	lines = append(lines,
		"  "+labelStyle.Render("Scanned: ")+valueStyle.Render(fmt.Sprintf("%d files across %d directories", r.TotalFiles, len(r.ScannedDirs))),
		"  "+labelStyle.Render("Unique: ")+valueStyle.Render(fmt.Sprintf("%d files", r.UniqueFiles)),
		"  "+labelStyle.Render("Duplicate groups: ")+accentStyle.Render(fmt.Sprintf("%d", len(r.DuplicateGroups))),
		"  "+labelStyle.Render("Wasted space: ")+errorStyle.Render(dedup.FormatSize(r.WastedBytes)),
	)
	if r.CacheHits > 0 {
		lines = append(lines, "  "+labelStyle.Render("Cache hits: ")+successStyle.Render(fmt.Sprintf("%d", r.CacheHits)))
	}
	lines = append(lines, "")

	if len(r.DuplicateGroups) == 0 {
		lines = append(lines, "  "+successStyle.Render("No duplicates found."))
		return lines
	}

	lines = append(lines, "  "+secondaryStyle.Render("Duplicate groups:"))
	lines = append(lines, "  "+mutedStyle.Render("Press Enter on a group to resolve, or 'd' to re-scan."))
	lines = append(lines, "")

	for i, g := range r.DuplicateGroups {
		copies := len(g.Files)
		wasted := int64(copies-1) * g.Size
		lines = append(lines,
			"  "+accentStyle.Render(fmt.Sprintf("Group %d", i+1))+
				" ("+dedup.FormatSize(g.Size)+", "+
				fmt.Sprintf("%d copies", copies)+", "+
				dedup.FormatSize(wasted)+" wasted)",
		)
		for _, f := range g.Files {
			display := "    " + f
			avail := width - 4
			if avail < 10 {
				avail = 10
			}
			if len(display) > avail {
				display = truncateMiddle(display, avail)
			}
			lines = append(lines, "    "+mutedStyle.Render(display))
		}
		lines = append(lines, "")
	}

	return lines
}

func (m model) duplicateResolverLines(width int) []string {
	var lines []string
	groups := m.data.DedupScan.DuplicateGroups

	if m.dupMode == "select" {
		lines = append(lines, "")
		lines = append(lines, "  "+titleStyle.Render("Select a duplicate group to resolve"))
		lines = append(lines, "")

		for i, g := range groups {
			copies := len(g.Files)
			wasted := int64(copies-1) * g.Size
			label := fmt.Sprintf("Group %d", i+1)
			detail := fmt.Sprintf("(%s, %d copies, %s wasted)", dedup.FormatSize(g.Size), copies, dedup.FormatSize(wasted))

			if i == m.dupSelectedGrp {
				lines = append(lines, "  "+accentStyle.Render("> "+label+" "+detail))
			} else {
				lines = append(lines, "  "+valueStyle.Render("  "+label+" ")+mutedStyle.Render(detail))
			}
		}
	} else if m.dupMode == "file" {
		g := groups[m.dupSelectedGrp]
		lines = append(lines, "")
		lines = append(lines, "  "+titleStyle.Render(fmt.Sprintf("Group %d — select file to keep", m.dupSelectedGrp+1)))
		lines = append(lines, "  "+mutedStyle.Render(fmt.Sprintf("(%s, %d copies)", dedup.FormatSize(g.Size), len(g.Files))))
		lines = append(lines, "")

		for i, f := range g.Files {
			info, _ := os.Stat(f)
			modTime := ""
			if info != nil {
				modTime = info.ModTime().Format("2006-01-02 15:04")
			}
			if i == m.dupSelectedFile {
				lines = append(lines, "  "+successStyle.Render("> ")+valueStyle.Render(filepath.Base(f))+" "+mutedStyle.Render(modTime))
			} else {
				lines = append(lines, "    "+valueStyle.Render(filepath.Base(f))+" "+mutedStyle.Render(modTime))
			}
		}
	}

	return lines
}
