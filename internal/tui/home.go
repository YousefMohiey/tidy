package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) handleHomeKey(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.selectedAction > 0 {
			m.selectedAction--
		}
		return m, nil
	case "down", "j":
		if m.selectedAction < 5 {
			m.selectedAction++
		}
		return m, nil
	case "enter":
		return m.executeAction(m.selectedAction)
	case "o":
		return m.executeAction(0)
	case "p":
		return m.executeAction(1)
	case "d":
		return m.executeAction(2)
	case "u":
		return m.executeAction(3)
	case "w":
		return m.executeAction(4)
	case "c":
		return m.executeAction(5)
	case "e":
		m.browsingDir = true
		startPath := m.data.SourceDir
		if startPath == "" {
			if home, err := os.UserHomeDir(); err == nil {
				startPath = home
			} else {
				startPath = string(os.PathSeparator)
			}
		}
		if strings.HasPrefix(startPath, "~") {
			if home, err := os.UserHomeDir(); err == nil {
				startPath = filepath.Join(home, startPath[1:])
			}
		}
		m.browsePath = startPath
		m.browseEntries = loadBrowseEntries(m.browsePath)
		m.browseSelected = 0
		m.browseScroll = 0
		return m, nil
	}
	return m, nil
}

func (m model) homeLines(width int) []string {
	var lines []string

	if m.data.SourceDir == "" && !m.browsingDir {
		lines = append(lines,
			"",
			"  "+titleStyle.Render("Welcome to tidy!"),
			"",
			"  "+valueStyle.Render("tidy organizes your files into categorized folders automatically."),
			"  "+valueStyle.Render("Before doing anything, it shows you a preview of what will happen."),
			"",
			"  "+accentStyle.Render("Getting started:"),
			"    "+mutedStyle.Render("1.")+valueStyle.Render(" Press ")+accentStyle.Render("e")+valueStyle.Render(" to browse and select a directory"),
			"    "+mutedStyle.Render("2.")+valueStyle.Render(" Press ")+accentStyle.Render("p")+valueStyle.Render(" to preview what tidy would do (nothing moves)"),
			"    "+mutedStyle.Render("3.")+valueStyle.Render(" Press ")+accentStyle.Render("o")+valueStyle.Render(" to organize when you're ready"),
			"",
			"  "+mutedStyle.Render("Everything is reversible — use ")+accentStyle.Render("u")+mutedStyle.Render(" to undo any time."),
			"",
		)
		lines = append(lines, m.renderActionMenu(width)...)
		return lines
	}

	if m.browsingDir {
		lines = append(lines, "")
		pathLine := "  " + labelStyle.Render("Browsing: ") + valueStyle.Render(m.browsePath)
		hints := "  " + mutedStyle.Render("[s]")+valueStyle.Render("elect") + "  " + mutedStyle.Render("[g]")+valueStyle.Render("o to path") + "  " + mutedStyle.Render("[esc]")+valueStyle.Render(" cancel")
		avail := width - lipgloss.Width(pathLine) - lipgloss.Width(hints)
		if avail < 1 {
			avail = 1
		}
		lines = append(lines, pathLine+strings.Repeat(" ", avail)+hints)
		if m.typingPath {
			p := m.pathInput
			c := m.pathCursor
			before := p[:c]
			at := "_"
			if c < len(p) {
				at = p[c : c+1]
			}
			after := ""
			if c+1 <= len(p) {
				after = p[c+1:]
			}
			lines = append(lines, "  "+accentStyle.Render("Go to: ")+
				valueStyle.Render(before)+cursorStyle.Render(at)+valueStyle.Render(after))
		}
		lines = append(lines, "")

		const maxVisible = 10
		boxWidth := width - 4
		if boxWidth < 30 {
			boxWidth = 30
		}
		innerRuler := strings.Repeat("─", boxWidth-2)
		lines = append(lines, "  "+borderStyle.Render("┌"+innerRuler+"┐"))

		totalEntries := len(m.browseEntries)
		end := m.browseScroll + maxVisible
		if end > totalEntries {
			end = totalEntries
		}

		if m.browseScroll > 0 {
			upHint := mutedStyle.Render(fmt.Sprintf("  ▲ %d more above", m.browseScroll))
			pad := boxWidth - 2 - lipgloss.Width(upHint)
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, "  "+borderStyle.Render("│")+upHint+strings.Repeat(" ", pad)+borderStyle.Render("│"))
		}

		for i := m.browseScroll; i < end; i++ {
			name := m.browseEntries[i]
			indicator := "  "
			var styled string
			displayName := name
			if name == "." {
				displayName = ". (select this folder)"
			} else if name == ".." {
				displayName = "← .. (go back)"
			} else if runtime.GOOS == "windows" && len(name) == 2 && name[1] == ':' {
				displayName = "[" + name + "]"
			}
			if i == m.browseSelected {
				indicator = accentStyle.Render("> ")
				styled = accentStyle.Render(displayName)
			} else {
				switch name {
				case ".":
					styled = successStyle.Render(displayName)
				case "..":
					styled = secondaryStyle.Render(displayName)
				default:
					if runtime.GOOS == "windows" && len(name) == 2 && name[1] == ':' {
						styled = secondaryStyle.Render(displayName)
					} else {
						styled = valueStyle.Render(displayName)
					}
				}
			}
			entry := indicator + styled
			contentW := lipgloss.Width(entry)
			pad := boxWidth - 2 - contentW - 1
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, "  "+borderStyle.Render("│")+" "+entry+strings.Repeat(" ", pad)+borderStyle.Render("│"))
		}

		if end < totalEntries {
			downHint := mutedStyle.Render(fmt.Sprintf("  ▼ %d more below", totalEntries-end))
			pad := boxWidth - 2 - lipgloss.Width(downHint)
			if pad < 0 {
				pad = 0
			}
			lines = append(lines, "  "+borderStyle.Render("│")+downHint+strings.Repeat(" ", pad)+borderStyle.Render("│"))
		}

		lines = append(lines, "  "+borderStyle.Render("└"+innerRuler+"┘"))
		lines = append(lines, "")
		lines = append(lines, m.renderActionMenu(width)...)
		return lines
	}

	src := m.data.SourceDir
	if src == "" {
		src = "N/A"
	}

	lines = append(lines,
		"  "+labelStyle.Render("Directory: ")+valueStyle.Render(src)+"    "+mutedStyle.Render("[e]dit"),
	)

	if m.data.Journal != nil {
		j := m.data.Journal
		lines = append(lines,
			"  "+labelStyle.Render("Last organized: ")+valueStyle.Render(j.Timestamp.Format("2006-01-02 15:04")),
			"  "+labelStyle.Render("Operations: ")+accentStyle.Render(fmt.Sprintf("%d", len(j.Operations))),
		)
	} else {
		lines = append(lines,
			"  "+labelStyle.Render("Last organized: ")+mutedStyle.Render("never"),
		)
	}

	lines = append(lines, "")

	if m.treePreview != nil {
		lines = append(lines, m.renderTreePreview(width)...)
		lines = append(lines, "")
	}

	lines = append(lines, m.renderActionMenu(width)...)

	return lines
}

func (m model) renderTreePreview(width int) []string {
	if m.treePreview == nil {
		return nil
	}
	tree := m.treePreview
	var lines []string

	var cats []string
	for cat := range tree.Categories {
		cats = append(cats, cat)
	}
	sort.Strings(cats)

	for _, cat := range cats {
		cp := tree.Categories[cat]
		count := len(cp.Files)
		if count == 0 {
			continue
		}
		lines = append(lines, "  "+secondaryStyle.Render("📁 "+cat)+" "+mutedStyle.Render(fmt.Sprintf("(%d files)", count)))
		for _, f := range cp.Files {
			lines = append(lines, "      "+arrowStyle.Render("→")+" "+valueStyle.Render(f))
		}
	}

	return lines
}

func (m model) renderActionMenu(width int) []string {
	type action struct {
		label    string
		shortcut string
	}
	actions := []action{
		{"Organize files", "o"},
		{"Preview (dry-run)", "p"},
		{"Scan for duplicates", "d"},
		{"Undo last operation", "u"},
		{"Toggle watch mode", "w"},
		{"Toggle context menu", "c"},
	}

	var lines []string

	titlePart := "─ Actions "
	remaining := width - 4 - lipgloss.Width(titlePart)
	if remaining < 0 {
		remaining = 0
	}
	topBorder := "  " + borderStyle.Render("┌"+titlePart+strings.Repeat("─", remaining)+"┐")
	lines = append(lines, topBorder)

	for i, a := range actions {
		indicator := "  "
		labelText := a.label
		shortcutText := mutedStyle.Render(fmt.Sprintf("[%s]", a.shortcut))

		if i == m.selectedAction {
			indicator = accentStyle.Render("> ")
			labelText = accentStyle.Render(a.label)
		} else {
			labelText = valueStyle.Render(a.label)
		}

		extra := ""
		if i == 4 && m.watching {
			extra = " " + successStyle.Render("(active)")
		}
		if i == 5 && runtime.GOOS == "windows" {
			if m.contextMenuInstalled() {
				extra = " " + successStyle.Render("(installed)")
			} else {
				extra = " " + mutedStyle.Render("(not installed)")
			}
		}

		content := indicator + labelText + extra
		contentW := lipgloss.Width(content)
		shortcutW := lipgloss.Width(shortcutText)
		padding := width - 6 - contentW - shortcutW
		if padding < 1 {
			padding = 1
		}
		line := "  " + borderStyle.Render("│") + " " + content + strings.Repeat(" ", padding) + shortcutText + " " + borderStyle.Render("│")
		lines = append(lines, line)
	}

	bottomRuler := width - 4
	if bottomRuler < 0 {
		bottomRuler = 0
	}
	bottomBorder := "  " + borderStyle.Render("└"+strings.Repeat("─", bottomRuler)+"┘")
	lines = append(lines, bottomBorder)

	return lines
}
