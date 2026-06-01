package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/YousefMohiey/tidy/internal/config"
	"github.com/YousefMohiey/tidy/internal/dedup"
	"github.com/YousefMohiey/tidy/internal/organizer"
	"github.com/YousefMohiey/tidy/internal/paths"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DashboardData struct {
	Journal   *organizer.Journal
	DedupScan *dedup.ScanResult
	SourceDir string
	Config    *config.Config
}

func Run(data DashboardData) error {
	m := newModel(data)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type ActionResult struct {
	Action    string
	Timestamp time.Time
	Success   bool
	Summary   string
	Details   []string
	Moves     []organizer.MoveRecord
}

type model struct {
	data             DashboardData
	activeTab        int
	scrollY          int
	width            int
	height           int
	status           string
	statusStyle      lipgloss.Style
	lastResult       *ActionResult
	selectedAction   int
	browsingDir      bool
	browsePath       string
	browseEntries    []string
	browseSelected   int
	browseScroll     int
	typingPath       bool
	pathInput        string
	pathCursor       int
	confirming       bool
	confirmMsg       string
	confirmAction    string
	watching         bool
	watchCancel      context.CancelFunc
	treePreview      *organizer.TreePreview
	showingPreview   bool
	progress         organizer.Progress
	dedupProgress    dedup.Progress
	dupSelectedGrp   int
	dupSelectedFile  int
	dupMode          string
	contextMenuState bool
	undoHistory      []UndoEntry
	undoSelected     int
}

func newModel(data DashboardData) model {
	m := model{
		data:        data,
		status:      "Ready",
		statusStyle: valueStyle,
	}
	if m.data.SourceDir == "" {
		if wd, err := os.Getwd(); err == nil {
			m.data.SourceDir = wd
		}
	}
	m.contextMenuState = m.contextMenuInstalled()
	m.undoHistory = loadUndoHistory()
	return m
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case organizeProgressMsg:
		m.progress = msg.progress
		m.status = fmt.Sprintf("%s: %d/%d (%s)", msg.progress.Status, msg.progress.FilesProcessed, msg.progress.FilesTotal, msg.progress.CurrentFile)
		m.statusStyle = accentStyle
		return m, nil

	case organizeResultMsg:
		m.status = "Ready"
		m.statusStyle = valueStyle
		m.progress = organizer.Progress{}
		if msg.err != nil {
			m.status = "Error: " + msg.err.Error()
			m.statusStyle = errorStyle
			m.lastResult = &ActionResult{Action: actionLabel(msg.dryRun), Timestamp: time.Now(), Success: false, Summary: msg.err.Error()}
		} else {
			r := msg.result
			m.lastResult = &ActionResult{
				Action:    actionLabel(msg.dryRun),
				Timestamp: time.Now(),
				Success:   true,
				Summary:   fmt.Sprintf("%d files moved, %d skipped, %d errors", r.FilesMoved, r.FilesSkipped, len(r.Errors)),
				Details:   formatMoves(r.Moves, m.data.SourceDir),
				Moves:     r.Moves,
			}
			if !msg.dryRun && len(r.Moves) > 0 {
				entry := UndoEntry{
					Timestamp:   time.Now(),
					FilesMoved:  r.FilesMoved,
					SourceDir:   m.data.SourceDir,
					JournalPath: paths.JournalPath(),
				}
				m.undoHistory = append([]UndoEntry{entry}, m.undoHistory...)
				if len(m.undoHistory) > 5 {
					m.undoHistory = m.undoHistory[:5]
				}
				m.undoSelected = 0
			}
			m.activeTab = 1
			m.scrollY = 0
			m.reloadJournal()
			return m, nil
		}

	case dedupProgressMsg:
		m.dedupProgress = msg.progress
		m.status = fmt.Sprintf("Scanning: %d/%d hashed (%d cached)", msg.progress.FilesScanned, msg.progress.FilesTotal, msg.progress.CacheHits)
		m.statusStyle = accentStyle
		return m, nil

	case dedupResultMsg:
		m.status = "Ready"
		m.statusStyle = valueStyle
		m.dedupProgress = dedup.Progress{}
		if msg.err != nil {
			m.status = "Error: " + msg.err.Error()
			m.statusStyle = errorStyle
		} else {
			m.data.DedupScan = msg.result
			m.activeTab = 2
			m.scrollY = 0
			m.status = fmt.Sprintf("Scan complete: %d duplicates, %d cache hits", len(msg.result.DuplicateGroups), msg.result.CacheHits)
			m.statusStyle = successStyle
		}
		return m, nil

	case undoResultMsg:
		m.status = "Ready"
		m.statusStyle = valueStyle
		if msg.err != nil {
			m.status = "Undo failed: " + msg.err.Error()
			m.statusStyle = errorStyle
			m.lastResult = &ActionResult{Action: "Undo", Timestamp: time.Now(), Success: false, Summary: msg.err.Error()}
		} else {
			m.lastResult = &ActionResult{Action: "Undo", Timestamp: time.Now(), Success: true, Summary: fmt.Sprintf("Restored %d files", msg.restored)}
			m.activeTab = 1
			m.scrollY = 0
			m.status = fmt.Sprintf("Undo complete: %d files restored", msg.restored)
			m.statusStyle = successStyle
		}
		m.reloadJournal()
		m.undoHistory = loadUndoHistory()
		m.undoSelected = 0
		return m, nil

	case watchDoneMsg:
		m.watching = false
		m.status = "Watch mode stopped"
		m.statusStyle = mutedStyle
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "ctrl+c" {
		if m.watching && m.watchCancel != nil {
			m.watchCancel()
		}
		return m, tea.Quit
	}

	if m.confirming {
		return m.handleConfirmKey(key)
	}

	if m.browsingDir {
		return m.handleBrowseKey(key)
	}

	switch key {
	case "q", "esc":
		if m.watching && m.watchCancel != nil {
			m.watchCancel()
		}
		return m, tea.Quit
	case "1":
		m.activeTab = 0
		m.scrollY = 0
		m.dupMode = ""
		return m, nil
	case "2":
		m.activeTab = 1
		m.scrollY = 0
		m.dupMode = ""
		return m, nil
	case "3":
		m.activeTab = 2
		m.scrollY = 0
		return m, nil
	case "4":
		m.activeTab = 3
		m.scrollY = 0
		m.dupMode = ""
		return m, nil
	case "tab":
		m.activeTab = (m.activeTab + 1) % 4
		m.scrollY = 0
		m.dupMode = ""
		return m, nil
	case "shift+tab":
		m.activeTab = (m.activeTab + 3) % 4
		m.scrollY = 0
		m.dupMode = ""
		return m, nil
	}

	switch m.activeTab {
	case 0:
		return m.handleHomeKey(msg, key)
	case 2:
		if m.dupMode != "" {
			return m.handleDupKey(key)
		}
		if key == "d" {
			return m.runDedupScan()
		}
		if key == "enter" && m.data.DedupScan != nil && len(m.data.DedupScan.DuplicateGroups) > 0 {
			m.dupMode = "select"
			m.dupSelectedGrp = 0
			m.dupSelectedFile = 0
			m.scrollY = 0
			return m, nil
		}
		return m.handleScrollKey(key)
	case 1:
		return m.handleScrollKey(key)
	case 3:
		return m.handleDetailsKey(key)
	}

	return m, nil
}

func (m model) handleConfirmKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "y", "Y":
		m.confirming = false
		action := m.confirmAction
		m.confirmAction = ""
		if action == "undo" {
			return m.runUndoLast()
		}
		if action == "install-context-menu" {
			return m.installContextMenu()
		}
		if action == "remove-context-menu" {
			return m.removeContextMenu()
		}
		if action == "undo-entry" {
			return m.runUndoEntry(m.undoSelected)
		}
	case "n", "N", "esc":
		m.confirming = false
		m.confirmAction = ""
	}
	return m, nil
}

func (m model) handleScrollKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "j", "down":
		m.scrollY++
	case "k", "up":
		if m.scrollY > 0 {
			m.scrollY--
		}
	}
	return m, nil
}

func (m model) executeAction(action int) (tea.Model, tea.Cmd) {
	switch action {
	case 0:
		return m.runOrganize(false)
	case 1:
		return m.runOrganize(true)
	case 2:
		return m.runDedupScan()
	case 3:
		return m.requestUndo()
	case 4:
		return m.toggleWatch()
	case 5:
		return m.toggleContextMenu()
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	innerWidth := m.width - 2
	if innerWidth < 40 {
		innerWidth = 40
	}

	contentHeight := m.height - 5
	if contentHeight < 1 {
		contentHeight = 1
	}

	var lines []string
	if m.confirming {
		lines = m.confirmLines(innerWidth)
	} else {
		lines = m.tabContent(innerWidth)
	}

	lines = append(lines, "")
	lines = append(lines, m.keyHints())

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
		} else {
			visible[i] = ""
		}
	}

	var sb strings.Builder
	b := borderStyle
	ruler := strings.Repeat("─", innerWidth)

	sb.WriteString("\x1b[2J\x1b[H")

	sb.WriteString(b.Render("╭" + ruler + "╮"))
	sb.WriteByte('\n')

	header := m.renderHeader(innerWidth)
	sb.WriteString(b.Render("│") + padLine(header, innerWidth) + b.Render("│"))
	sb.WriteByte('\n')

	sb.WriteString(b.Render("├" + ruler + "┤"))
	sb.WriteByte('\n')

	for _, line := range visible {
		sb.WriteString(b.Render("│") + padLine(line, innerWidth) + b.Render("│"))
		sb.WriteByte('\n')
	}

	statusLine := m.renderStatusBar(innerWidth)
	sb.WriteString(b.Render("│") + padLine(statusLine, innerWidth) + b.Render("│"))
	sb.WriteByte('\n')

	sb.WriteString(b.Render("╰" + ruler + "╯"))
	sb.WriteByte('\n')

	return sb.String()
}

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
	names := []string{"Home", "Results", "Duplicates", "Details"}
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

func (m model) renderStatusBar(width int) string {
	statusLabel := labelStyle.Render("  Status: ")
	statusVal := m.statusStyle.Render(m.status)
	return statusLabel + statusVal
}

func (m model) keyHints() string {
	if m.confirming {
		return "  " + mutedStyle.Render("y confirm  n cancel")
	}
	if m.typingPath {
		return "  " + mutedStyle.Render("\u2190\u2192 move cursor  \u23ce confirm  esc cancel")
	}
	if m.browsingDir {
		return "  " + mutedStyle.Render("\u2191\u2193 navigate  \u23ce select  g type path  esc cancel")
	}
	if m.showingPreview {
		return "  " + mutedStyle.Render("\u23ce organize  esc back")
	}
	if m.dupMode == "select" {
		return "  " + mutedStyle.Render("\u2191\u2193 select  \u23ce open  D delete all  esc back")
	}
	if m.dupMode == "file" {
		return "  " + mutedStyle.Render("\u2191\u2193 select  \u23ce keep this  esc back")
	}
	switch m.activeTab {
	case 0:
		return "  " + mutedStyle.Render("\u2191\u2193 navigate  \u23ce execute  e set dir  q quit")
	case 1:
		return "  " + mutedStyle.Render("\u2191\u2193 scroll  tab switch")
	case 2:
		return "  " + mutedStyle.Render("d scan  \u2191\u2193 select  tab switch")
	case 3:
		return "  " + mutedStyle.Render("\u2191\u2193 select  \u23ce undo  tab switch")
	}
	return ""
}

func (m model) tabContent(width int) []string {
	switch m.activeTab {
	case 0:
		return m.homeLines(width)
	case 1:
		return m.resultsLines(width)
	case 2:
		return m.duplicatesLines(width)
	default:
		return m.detailsLines(width)
	}
}

func (m model) confirmLines(width int) []string {
	var lines []string

	for i := 0; i < 3; i++ {
		lines = append(lines, "")
	}

	boxWidth := 50
	if boxWidth > width-4 {
		boxWidth = width - 4
	}
	if boxWidth < 30 {
		boxWidth = 30
	}

	indent := (width - boxWidth) / 2
	if indent < 2 {
		indent = 2
	}
	pad := strings.Repeat(" ", indent)

	innerRuler := strings.Repeat("─", boxWidth-2)
	lines = append(lines, pad+borderStyle.Render("┌"+innerRuler+"┐"))

	titleContent := "  Confirm"
	titleLine := pad + borderStyle.Render("│") + titleStyle.Render(titleContent) + strings.Repeat(" ", boxWidth-2-lipgloss.Width(titleContent)) + borderStyle.Render("│")
	lines = append(lines, titleLine)

	lines = append(lines, pad+borderStyle.Render("│")+strings.Repeat(" ", boxWidth-2)+borderStyle.Render("│"))

	msgContent := "  " + m.confirmMsg
	if lipgloss.Width(msgContent) > boxWidth-2 {
		msgContent = truncateMiddle(msgContent, boxWidth-2)
	}
	msgPad := boxWidth - 2 - lipgloss.Width(msgContent)
	if msgPad < 0 {
		msgPad = 0
	}
	lines = append(lines, pad+borderStyle.Render("│")+valueStyle.Render(msgContent)+strings.Repeat(" ", msgPad)+borderStyle.Render("│"))

	lines = append(lines, pad+borderStyle.Render("│")+strings.Repeat(" ", boxWidth-2)+borderStyle.Render("│"))

	buttons := "  " + successStyle.Render("[y] Confirm") + "   " + errorStyle.Render("[n] Cancel")
	btnPad := boxWidth - 2 - lipgloss.Width(buttons)
	if btnPad < 0 {
		btnPad = 0
	}
	lines = append(lines, pad+borderStyle.Render("│")+buttons+strings.Repeat(" ", btnPad)+borderStyle.Render("│"))

	lines = append(lines, pad+borderStyle.Render("└"+innerRuler+"┘"))

	return lines
}

func padLine(s string, width int) string {
	visible := lipgloss.Width(s)
	if visible >= width {
		return s
	}
	return s + strings.Repeat(" ", width-visible)
}

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
