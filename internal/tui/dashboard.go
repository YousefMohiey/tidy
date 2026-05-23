package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/verhafter/tidy/internal/dedup"
	"github.com/verhafter/tidy/internal/organizer"
)

// ── Colors ──────────────────────────────────────────────────────────────────

var (
	colorPrimary   = lipgloss.Color("#7C3AED")
	colorSecondary = lipgloss.Color("#06B6D4")
	colorAccent    = lipgloss.Color("#F59E0B")
	colorText      = lipgloss.Color("#E2E8F0")
	colorMuted     = lipgloss.Color("#64748B")
	colorBorder    = lipgloss.Color("#334155")
)

// ── Styles ──────────────────────────────────────────────────────────────────

var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	activeTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
	inactiveTabStyle = lipgloss.NewStyle().Foreground(colorMuted)
	labelStyle       = lipgloss.NewStyle().Foreground(colorMuted)
	valueStyle       = lipgloss.NewStyle().Foreground(colorText)
	accentStyle      = lipgloss.NewStyle().Foreground(colorAccent)
	secondaryStyle   = lipgloss.NewStyle().Foreground(colorSecondary)
	mutedStyle       = lipgloss.NewStyle().Foreground(colorMuted)
	arrowStyle       = lipgloss.NewStyle().Foreground(colorAccent)
	borderStyle      = lipgloss.NewStyle().Foreground(colorBorder)
)

// ── Public API ──────────────────────────────────────────────────────────────

// DashboardData holds all data the dashboard needs to display.
type DashboardData struct {
	Journal   *organizer.Journal // may be nil
	DedupScan *dedup.ScanResult  // may be nil
	SourceDir string
}

// NewDashboard creates and returns a Bubble Tea program for the dashboard.
func NewDashboard(data DashboardData) *tea.Program {
	return tea.NewProgram(
		model{data: data},
		tea.WithAltScreen(),
	)
}

// Run starts the dashboard TUI (blocking).
func Run(data DashboardData) error {
	_, err := NewDashboard(data).Run()
	return err
}

// ── Model ───────────────────────────────────────────────────────────────────

type model struct {
	data      DashboardData
	activeTab int // 0=Overview, 1=Duplicates, 2=Help
	scrollY   int // vertical scroll offset for content
	width     int // terminal width
	height    int // terminal height
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "1":
			m.activeTab = 0
			m.scrollY = 0
		case "2":
			m.activeTab = 1
			m.scrollY = 0
		case "3":
			m.activeTab = 2
			m.scrollY = 0
		case "tab":
			m.activeTab = (m.activeTab + 1) % 3
			m.scrollY = 0
		case "shift+tab":
			m.activeTab = (m.activeTab + 2) % 3
			m.scrollY = 0
		case "j", "down":
			m.scrollY++
		case "k", "up":
			if m.scrollY > 0 {
				m.scrollY--
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Inner width: terminal width minus left+right border characters
	innerWidth := m.width - 2
	if innerWidth < 38 {
		innerWidth = 38
	}

	// Content height: terminal height minus 5 fixed lines
	// (top-border, header, separator, bottom-border, footer)
	contentHeight := m.height - 5
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Generate content lines for active tab
	lines := m.tabContent(innerWidth)

	// Clamp scroll to valid range
	maxScroll := len(lines) - contentHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	scrollY := m.scrollY
	if scrollY > maxScroll {
		scrollY = maxScroll
	}
	if scrollY < 0 {
		scrollY = 0
	}

	// Extract visible window
	start := scrollY
	end := start + contentHeight
	if end > len(lines) {
		end = len(lines)
	}
	visible := make([]string, contentHeight)
	for i := 0; i < contentHeight; i++ {
		idx := start + i
		if idx < end {
			visible[i] = lines[idx]
		}
	}

	// Build the full view
	var sb strings.Builder
	b := borderStyle
	ruler := strings.Repeat("─", innerWidth)

	// Top border
	sb.WriteString(b.Render("╭" + ruler + "╮"))
	sb.WriteByte('\n')

	// Header line
	header := m.renderHeader(innerWidth)
	sb.WriteString(b.Render("│") + padLine(header, innerWidth) + b.Render("│"))
	sb.WriteByte('\n')

	// Separator
	sb.WriteString(b.Render("├" + ruler + "┤"))
	sb.WriteByte('\n')

	// Content lines
	for _, line := range visible {
		sb.WriteString(b.Render("│") + padLine(line, innerWidth) + b.Render("│"))
		sb.WriteByte('\n')
	}

	// Bottom border
	sb.WriteString(b.Render("╰" + ruler + "╯"))
	sb.WriteByte('\n')

	// Footer (outside border)
	sb.WriteString(m.renderFooter())

	return sb.String()
}

// ── Header ──────────────────────────────────────────────────────────────────

func (m model) renderHeader(width int) string {
	title := titleStyle.Render("  tidy dashboard")
	tabs := m.renderTabs()

	titleW := lipgloss.Width(title)
	tabsW := lipgloss.Width(tabs)
	gap := width - titleW - tabsW
	if gap < 1 {
		gap = 1
	}
	return title + strings.Repeat(" ", gap) + tabs
}

func (m model) renderTabs() string {
	names := []string{"Overview", "Duplicates", "Help"}
	var parts []string
	for i, name := range names {
		label := fmt.Sprintf("[%d] %s", i+1, name)
		if i == m.activeTab {
			parts = append(parts, activeTabStyle.Render(label))
		} else {
			parts = append(parts, inactiveTabStyle.Render(label))
		}
	}
	return strings.Join(parts, " ")
}

// ── Footer ──────────────────────────────────────────────────────────────────

func (m model) renderFooter() string {
	hints := []string{
		mutedStyle.Render("tab/shift+tab") + valueStyle.Render(": switch tabs"),
		mutedStyle.Render("j/k") + valueStyle.Render(": scroll"),
		mutedStyle.Render("q") + valueStyle.Render(": quit"),
	}
	return "  " + strings.Join(hints, "  ")
}

// ── Tab Content Router ──────────────────────────────────────────────────────

func (m model) tabContent(width int) []string {
	switch m.activeTab {
	case 0:
		return m.overviewLines(width)
	case 1:
		return m.duplicatesLines(width)
	default:
		return m.helpLines()
	}
}

// ── Tab 1: Overview ─────────────────────────────────────────────────────────

func (m model) overviewLines(width int) []string {
	var lines []string

	// Resolve source directory
	src := m.data.SourceDir
	if src == "" && m.data.Journal != nil {
		src = m.data.Journal.SourceDir
	}
	if src == "" {
		src = "N/A"
	}

	lines = append(lines,
		"  "+labelStyle.Render("Source:     ")+valueStyle.Render(src),
	)

	if m.data.Journal == nil {
		lines = append(lines,
			"",
			"  "+mutedStyle.Render("No operations recorded yet."),
		)
		return lines
	}

	j := m.data.Journal

	lines = append(lines,
		"  "+labelStyle.Render("Last run:   ")+valueStyle.Render(j.Timestamp.Format("2006-01-02 15:04:05")),
		"  "+labelStyle.Render("Operations: ")+accentStyle.Render(fmt.Sprintf("%d", len(j.Operations))),
	)

	if len(j.Operations) == 0 {
		lines = append(lines, "", "  "+mutedStyle.Render("No moves in this journal."))
		return lines
	}

	lines = append(lines, "", "  "+secondaryStyle.Render("Recent moves:"))

	// Find max source filename length for alignment
	ops := j.Operations
	maxSrcLen := 0
	for _, op := range ops {
		name := filepath.Base(op.Source)
		if len(name) > maxSrcLen {
			maxSrcLen = len(name)
		}
	}
	if maxSrcLen > 30 {
		maxSrcLen = 30
	}

	// Render operations in reverse chronological order
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		srcName := filepath.Base(op.Source)
		if len(srcName) > 30 {
			srcName = truncateMiddle(srcName, 30)
		}

		// Make destination relative to source dir
		dst := op.Destination
		if j.SourceDir != "" {
			if rel, err := filepath.Rel(j.SourceDir, dst); err == nil {
				dst = rel
			}
		}

		// Calculate available width for destination
		// indent(4) + padded src name + space + arrow + space = 4 + maxSrcLen + 3
		used := 4 + maxSrcLen + 3
		avail := width - used
		if avail < 10 {
			avail = 10
		}
		if lipgloss.Width(dst) > avail {
			dst = truncateMiddle(dst, avail)
		}

		line := fmt.Sprintf("    %-*s", maxSrcLen, srcName) +
			" " + arrowStyle.Render("→") + " " + valueStyle.Render(dst)
		lines = append(lines, line)
	}

	return lines
}

// ── Tab 2: Duplicates ───────────────────────────────────────────────────────

func (m model) duplicatesLines(width int) []string {
	var lines []string

	if m.data.DedupScan == nil {
		lines = append(lines,
			"",
			"  "+mutedStyle.Render("No duplicate scan available."),
			"  "+mutedStyle.Render("Run 'tidy dedup <dir>' first."),
		)
		return lines
	}

	s := m.data.DedupScan

	lines = append(lines,
		"  "+labelStyle.Render("Scan: ")+valueStyle.Render(
			fmt.Sprintf("%d files, %d duplicate groups", s.TotalFiles, len(s.DuplicateGroups)),
		),
		"  "+labelStyle.Render("Wasted space: ")+accentStyle.Render(dedup.FormatSize(s.WastedBytes)),
		"",
	)

	if len(s.DuplicateGroups) == 0 {
		lines = append(lines, "  "+secondaryStyle.Render("No duplicates found. Your files are clean!"))
		return lines
	}

	for i, g := range s.DuplicateGroups {
		header := fmt.Sprintf("  Group %d (%s, %d copies):",
			i+1, dedup.FormatSize(g.Size), len(g.Files))
		lines = append(lines, secondaryStyle.Render(header))

		for _, f := range g.Files {
			display := f
			avail := width - 6 // indent(4) + margin(2)
			if avail < 10 {
				avail = 10
			}
			if len(display) > avail {
				display = truncateMiddle(display, avail)
			}
			lines = append(lines, "    "+mutedStyle.Render(display))
		}
		lines = append(lines, "") // blank line between groups
	}

	return lines
}

// ── Tab 3: Help ─────────────────────────────────────────────────────────────

func (m model) helpLines() []string {
	return []string{
		"  " + secondaryStyle.Render("Keyboard shortcuts:"),
		"    " + mutedStyle.Render("1, 2, 3     ") + valueStyle.Render("Switch tabs"),
		"    " + mutedStyle.Render("Tab         ") + valueStyle.Render("Next tab"),
		"    " + mutedStyle.Render("Shift+Tab   ") + valueStyle.Render("Previous tab"),
		"    " + mutedStyle.Render("j / ↓       ") + valueStyle.Render("Scroll down"),
		"    " + mutedStyle.Render("k / ↑       ") + valueStyle.Render("Scroll up"),
		"    " + mutedStyle.Render("q / Esc     ") + valueStyle.Render("Quit"),
		"",
		"  " + secondaryStyle.Render("Commands:"),
		"    " + mutedStyle.Render("tidy organize <dir>     ") + valueStyle.Render("Organize files"),
		"    " + mutedStyle.Render("tidy organize --dry-run ") + valueStyle.Render("Preview changes"),
		"    " + mutedStyle.Render("tidy watch <dir>        ") + valueStyle.Render("Auto-organize"),
		"    " + mutedStyle.Render("tidy undo               ") + valueStyle.Render("Rollback"),
		"    " + mutedStyle.Render("tidy dedup <dir>        ") + valueStyle.Render("Find duplicates"),
		"    " + mutedStyle.Render("tidy dashboard          ") + valueStyle.Render("This dashboard"),
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// padLine pads s with trailing spaces to fill the given visible width.
func padLine(s string, width int) string {
	visible := lipgloss.Width(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

// truncateMiddle shortens s to maxLen by inserting "..." in the center.
func truncateMiddle(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 5 {
		if maxLen <= 0 {
			return ""
		}
		return s[:maxLen]
	}
	ellipsis := "..."
	remaining := maxLen - len(ellipsis)
	front := remaining / 2
	back := remaining - front
	return s[:front] + ellipsis + s[len(s)-back:]
}
