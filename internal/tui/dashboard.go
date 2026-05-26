package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/verhafter/tidy/internal/config"
	"github.com/verhafter/tidy/internal/dedup"
	"github.com/verhafter/tidy/internal/organizer"
	"github.com/verhafter/tidy/internal/paths"
	"github.com/verhafter/tidy/internal/watcher"
)

// ── Colors ──────────────────────────────────────────────────────────────────

var (
	colorPrimary   = lipgloss.Color("#7C3AED")
	colorSecondary = lipgloss.Color("#06B6D4")
	colorAccent    = lipgloss.Color("#F59E0B")
	colorText      = lipgloss.Color("#E2E8F0")
	colorMuted     = lipgloss.Color("#64748B")
	colorBorder    = lipgloss.Color("#334155")
	colorSuccess   = lipgloss.Color("#22C55E")
	colorError     = lipgloss.Color("#EF4444")
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
	successStyle     = lipgloss.NewStyle().Foreground(colorSuccess)
	errorStyle       = lipgloss.NewStyle().Foreground(colorError)
)

// ── Public API ──────────────────────────────────────────────────────────────

// DashboardData holds all data the dashboard needs to display.
type DashboardData struct {
	Journal   *organizer.Journal // may be nil
	DedupScan *dedup.ScanResult  // may be nil
	SourceDir string
	Config    *config.Config // may be nil; needed for organize operations
}

// NewDashboard creates and returns a Bubble Tea program for the dashboard.
func NewDashboard(data DashboardData) *tea.Program {
	m := model{
		data:   data,
		status: "Ready",
	}
	if m.data.SourceDir == "" {
		if wd, err := os.Getwd(); err == nil {
			m.data.SourceDir = wd
		}
	}
	return tea.NewProgram(m, tea.WithAltScreen())
}

// Run starts the dashboard TUI (blocking).
func Run(data DashboardData) error {
	_, err := NewDashboard(data).Run()
	return err
}

// ── Async Result Messages ───────────────────────────────────────────────────

type organizeResultMsg struct {
	result *organizer.Result
	err    error
	dryRun bool
}

type dedupResultMsg struct {
	result *dedup.ScanResult
	err    error
}

type undoResultMsg struct {
	restored int
	err      error
}

type watchEventMsg struct {
	info string
}

type watchDoneMsg struct{}



// ── ActionResult ────────────────────────────────────────────────────────────

// ActionResult stores the outcome of a user-initiated operation.
type ActionResult struct {
	Action    string
	Timestamp time.Time
	Success   bool
	Summary   string
	Details   []string
}

// ── Model ───────────────────────────────────────────────────────────────────

type model struct {
	// Data
	data DashboardData

	// UI state
	activeTab int // 0=Home, 1=Results, 2=Duplicates, 3=Help
	scrollY   int
	width     int
	height    int

	// Folder browser
	browsingDir    bool
	browsePath     string
	browseEntries  []string
	browseSelected int
	browseScroll   int
	typingPath     bool
	pathInput      string

	// Action menu (Home tab)
	selectedAction int // 0=Organize, 1=Preview, 2=Dedup, 3=Undo, 4=Watch

	// Operation state
	status      string
	statusStyle lipgloss.Style
	lastResult  *ActionResult

	// Confirmation dialog
	confirming     bool
	confirmMsg     string
	confirmAction  string // "undo"

	// Watch mode
	watching    bool
	watchCancel context.CancelFunc
}

func (m model) Init() tea.Cmd {
	return nil
}

// ── Update ──────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case organizeResultMsg:
		m.status = "Ready"
		m.statusStyle = valueStyle
		if msg.err != nil {
			m.status = "Error: " + msg.err.Error()
			m.statusStyle = errorStyle
			m.lastResult = &ActionResult{
				Action:    actionLabel(msg.dryRun),
				Timestamp: time.Now(),
				Success:   false,
				Summary:   msg.err.Error(),
			}
		} else {
			r := msg.result
			m.lastResult = &ActionResult{
				Action:    actionLabel(msg.dryRun),
				Timestamp: time.Now(),
				Success:   true,
				Summary:   fmt.Sprintf("%d files moved, %d skipped, %d errors", r.FilesMoved, r.FilesSkipped, len(r.Errors)),
				Details:   formatMoves(r.Moves, m.data.SourceDir),
			}
			m.activeTab = 1 // switch to Results tab
			m.scrollY = 0
		}
		m.reloadJournal()
		return m, nil

	case dedupResultMsg:
		m.status = "Ready"
		m.statusStyle = valueStyle
		if msg.err != nil {
			m.status = "Error: " + msg.err.Error()
			m.statusStyle = errorStyle
		} else {
			m.data.DedupScan = msg.result
			m.activeTab = 2 // switch to Duplicates tab
			m.scrollY = 0
			m.status = fmt.Sprintf("Scan complete: %d duplicates found", len(msg.result.DuplicateGroups))
			m.statusStyle = successStyle
		}
		return m, nil

	case undoResultMsg:
		m.status = "Ready"
		m.statusStyle = valueStyle
		if msg.err != nil {
			m.status = "Undo failed: " + msg.err.Error()
			m.statusStyle = errorStyle
			m.lastResult = &ActionResult{
				Action:    "Undo",
				Timestamp: time.Now(),
				Success:   false,
				Summary:   msg.err.Error(),
			}
		} else {
			m.lastResult = &ActionResult{
				Action:    "Undo",
				Timestamp: time.Now(),
				Success:   true,
				Summary:   fmt.Sprintf("Restored %d files", msg.restored),
			}
			m.activeTab = 1
			m.scrollY = 0
			m.status = fmt.Sprintf("Undo complete: %d files restored", msg.restored)
			m.statusStyle = successStyle
		}
		m.reloadJournal()
		return m, nil

	case watchEventMsg:
		m.status = msg.info
		m.statusStyle = successStyle
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

// ── Key Handling ────────────────────────────────────────────────────────────

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Ctrl+C always quits
	if key == "ctrl+c" {
		if m.watching && m.watchCancel != nil {
			m.watchCancel()
		}
		return m, tea.Quit
	}

	// Confirmation dialog takes priority
	if m.confirming {
		return m.handleConfirmKey(key)
	}

	if m.browsingDir {
		return m.handleBrowseKey(key)
	}

	// Global keys
	switch key {
	case "q", "esc":
		if m.watching && m.watchCancel != nil {
			m.watchCancel()
		}
		return m, tea.Quit
	case "1":
		m.activeTab = 0
		m.scrollY = 0
		return m, nil
	case "2":
		m.activeTab = 1
		m.scrollY = 0
		return m, nil
	case "3":
		m.activeTab = 2
		m.scrollY = 0
		return m, nil
	case "4":
		m.activeTab = 3
		m.scrollY = 0
		return m, nil
	case "tab":
		m.activeTab = (m.activeTab + 1) % 4
		m.scrollY = 0
		return m, nil
	case "shift+tab":
		m.activeTab = (m.activeTab + 3) % 4
		m.scrollY = 0
		return m, nil
	}

	// Tab-specific keys
	switch m.activeTab {
	case 0: // Home
		return m.handleHomeKey(msg, key)
	case 2: // Duplicates — 'd' triggers scan
		if key == "d" {
			return m.runDedupScan()
		}
		return m.handleScrollKey(key)
	case 1, 3: // Results, Help
		return m.handleScrollKey(key)
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
			return m.runUndo()
		}
	case "n", "N", "esc":
		m.confirming = false
		m.confirmAction = ""
	}
	return m, nil
}

func (m model) handleBrowseKey(key string) (tea.Model, tea.Cmd) {
	if m.typingPath {
		return m.handlePathInputKey(key)
	}
	switch key {
	case "g":
		m.typingPath = true
		m.pathInput = m.browsePath
		return m, nil
	case "up", "k":
		if m.browseSelected > 0 {
			m.browseSelected--
		}
		m.adjustBrowseScroll()
		return m, nil
	case "down", "j":
		if m.browseSelected < len(m.browseEntries)-1 {
			m.browseSelected++
		}
		m.adjustBrowseScroll()
		return m, nil
	case "home":
		m.browseSelected = 0
		m.adjustBrowseScroll()
		return m, nil
	case "end":
		if len(m.browseEntries) > 0 {
			m.browseSelected = len(m.browseEntries) - 1
		}
		m.adjustBrowseScroll()
		return m, nil
	case "enter":
		if len(m.browseEntries) == 0 {
			return m, nil
		}
		name := m.browseEntries[m.browseSelected]
		if name == "." {
			m.data.SourceDir = m.browsePath
			m.browsingDir = false
			return m, nil
		}
		if name == ".." {
			m.browsePath = filepath.Dir(m.browsePath)
			if m.browsePath == "." {
				m.browsePath = string(os.PathSeparator)
			}
		} else if runtime.GOOS == "windows" && len(name) == 2 && name[1] == ':' {
			// Drive letter entry (e.g. "D:") — navigate to D:\
			m.browsePath = name + string(os.PathSeparator)
		} else {
			m.browsePath = filepath.Join(m.browsePath, name)
		}
		m.browseEntries = loadBrowseEntries(m.browsePath)
		m.browseSelected = 0
		m.browseScroll = 0
		return m, nil
	case "backspace":
		parent := filepath.Dir(m.browsePath)
		if parent != m.browsePath {
			m.browsePath = parent
			if m.browsePath == "." {
				m.browsePath = string(os.PathSeparator)
			}
			m.browseEntries = loadBrowseEntries(m.browsePath)
			m.browseSelected = 0
			m.browseScroll = 0
		}
		return m, nil
	case "s", "ctrl+s":
		m.data.SourceDir = m.browsePath
		m.browsingDir = false
		return m, nil
	case "esc":
		m.browsingDir = false
		return m, nil
	}
	return m, nil
}

func (m model) handlePathInputKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "enter":
		cleanPath := filepath.Clean(strings.TrimSpace(m.pathInput))
		if info, err := os.Stat(cleanPath); err == nil && info.IsDir() {
			m.browsePath = cleanPath
			m.browseEntries = loadBrowseEntries(m.browsePath)
			m.browseSelected = 0
			m.browseScroll = 0
		}
		m.typingPath = false
		m.pathInput = ""
		return m, nil
	case "esc":
		m.typingPath = false
		m.pathInput = ""
		return m, nil
	case "backspace":
		if len(m.pathInput) > 0 {
			m.pathInput = m.pathInput[:len(m.pathInput)-1]
		}
		return m, nil
	case "ctrl+u":
		m.pathInput = ""
		return m, nil
	default:
		for _, r := range key {
			if r >= 32 && r < 127 {
				m.pathInput += string(r)
			}
		}
		return m, nil
	}
}

func (m *model) adjustBrowseScroll() {
	const maxVisible = 10
	if m.browseSelected < m.browseScroll {
		m.browseScroll = m.browseSelected
	}
	if m.browseSelected >= m.browseScroll+maxVisible {
		m.browseScroll = m.browseSelected - maxVisible + 1
	}
}

// listWindowsDrives returns available drive letters on Windows (e.g. "C:", "D:").
// On non-Windows systems it returns nil.
func listWindowsDrives() []string {
	if runtime.GOOS != "windows" {
		return nil
	}
	var drives []string
	for l := 'A'; l <= 'Z'; l++ {
		path := string(l) + ":\\"
		if _, err := os.Stat(path); err == nil {
			drives = append(drives, string(l)+":")
		}
	}
	return drives
}

func loadBrowseEntries(path string) []string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		dirs = append(dirs, name)
	}
	sort.Strings(dirs)
	result := []string{"."}

	cleanPath := filepath.Clean(path)
	parent := filepath.Dir(cleanPath)
	canGoUp := parent != cleanPath
	if canGoUp {
		result = append(result, "..")
	} else if runtime.GOOS == "windows" {
		// At filesystem root on Windows — show available drives instead of ".."
		drives := listWindowsDrives()
		result = append(result, drives...)
	}

	result = append(result, dirs...)
	return result
}

func (m model) handleHomeKey(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.selectedAction > 0 {
			m.selectedAction--
		}
		return m, nil
	case "down", "j":
		if m.selectedAction < 4 {
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

// ── Action Execution ────────────────────────────────────────────────────────

func (m model) executeAction(action int) (tea.Model, tea.Cmd) {
	switch action {
	case 0: // Organize
		return m.runOrganize(false)
	case 1: // Preview
		return m.runOrganize(true)
	case 2: // Dedup scan
		return m.runDedupScan()
	case 3: // Undo
		return m.requestUndo()
	case 4: // Watch
		return m.toggleWatch()
	}
	return m, nil
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
		opts := organizer.Options{DryRun: dryRun}
		org := organizer.New(cfg, opts)
		result, err := org.Organize(dir)
		return organizeResultMsg{result: result, err: err, dryRun: dryRun}
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
		result, err := scanner.Scan(dir)
		return dedupResultMsg{result: result, err: err}
	}
}

func (m model) requestUndo() (tea.Model, tea.Cmd) {
	if m.data.Journal == nil {
		m.status = "No operations to undo"
		m.statusStyle = mutedStyle
		return m, nil
	}

	opCount := len(m.data.Journal.Operations)
	m.confirming = true
	m.confirmAction = "undo"
	m.confirmMsg = fmt.Sprintf("Undo last operation? (%d files will be moved back)", opCount)
	return m, nil
}

func (m model) runUndo() (tea.Model, tea.Cmd) {
	m.status = "Undoing..."
	m.statusStyle = accentStyle

	journal := m.data.Journal
	return m, func() tea.Msg {
		restored, err := journal.Undo()
		if err == nil {
			// Remove journal file after successful undo
			jPath := journalPath()
			if jPath != "" {
				_ = os.Remove(jPath)
			}
		}
		return undoResultMsg{restored: restored, err: err}
	}
}

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
		_ = w.Watch(ctx)
		return watchDoneMsg{}
	}
}

// ── Journal Helpers ─────────────────────────────────────────────────────────

func journalPath() string {
	return paths.JournalPath()
}

func (m *model) reloadJournal() {
	jPath := journalPath()
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

// ── View ────────────────────────────────────────────────────────────────────

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	innerWidth := m.width - 2
	if innerWidth < 40 {
		innerWidth = 40
	}

	// Layout: top-border(1) + header(1) + separator(1) + content(N) + status(1) + bottom-border(1)
	// Footer is rendered inside the content area to ensure proper clearing
	// Using -6 instead of -5 to leave 1 line of padding, preventing terminal scrollback from hiding the header
	contentHeight := m.height - 6
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Generate content lines
	var lines []string
	if m.confirming {
		lines = m.confirmLines(innerWidth)
	} else {
		lines = m.tabContent(innerWidth)
	}

	// Append footer to content lines so it's part of the scrollable area
	lines = append(lines, "")
	lines = append(lines, m.renderFooter())

	// Clamp scroll
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
		} else {
			visible[i] = "" // Fill remaining lines with empty strings
		}
	}

	var sb strings.Builder
	b := borderStyle
	ruler := strings.Repeat("─", innerWidth)

	// Top border
	sb.WriteString(b.Render("╭" + ruler + "╮"))
	sb.WriteByte('\n')

	// Header
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

	// Status bar
	statusLine := m.renderStatusBar(innerWidth)
	sb.WriteString(b.Render("│") + padLine(statusLine, innerWidth) + b.Render("│"))
	sb.WriteByte('\n')

	// Bottom border
	sb.WriteString(b.Render("╰" + ruler + "╯"))
	sb.WriteByte('\n')

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
	names := []string{"Home", "Results", "Duplicates", "Help"}
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

// ── Status Bar ──────────────────────────────────────────────────────────────

func (m model) renderStatusBar(width int) string {
	statusLabel := labelStyle.Render("  Status: ")
	statusVal := m.statusStyle.Render(m.status)
	return statusLabel + statusVal
}

// ── Footer ──────────────────────────────────────────────────────────────────

func (m model) renderFooter() string {
	if m.browsingDir {
		hints := []string{
			mutedStyle.Render("j/k") + valueStyle.Render(": navigate"),
			mutedStyle.Render("enter") + valueStyle.Render(": open"),
			mutedStyle.Render("s") + valueStyle.Render(": select"),
			mutedStyle.Render("backspace") + valueStyle.Render(": up"),
			mutedStyle.Render("esc") + valueStyle.Render(": cancel"),
		}
		return "  " + strings.Join(hints, "  ")
	}
	if m.confirming {
		hints := []string{
			mutedStyle.Render("y") + valueStyle.Render(": confirm"),
			mutedStyle.Render("n/esc") + valueStyle.Render(": cancel"),
		}
		return "  " + strings.Join(hints, "  ")
	}

	var hints []string
	hints = append(hints, mutedStyle.Render("1-4")+valueStyle.Render(": tabs"))

	switch m.activeTab {
	case 0: // Home
		hints = append(hints,
			mutedStyle.Render("enter")+valueStyle.Render(": select"),
			mutedStyle.Render("e")+valueStyle.Render(": browse dir"),
		)
	case 1: // Results
		hints = append(hints, mutedStyle.Render("j/k")+valueStyle.Render(": scroll"))
	case 2: // Duplicates
		hints = append(hints,
			mutedStyle.Render("d")+valueStyle.Render(": scan"),
			mutedStyle.Render("j/k")+valueStyle.Render(": scroll"),
		)
	}

	hints = append(hints, mutedStyle.Render("q")+valueStyle.Render(": quit"))
	return "  " + strings.Join(hints, "  ")
}

// ── Tab Content Router ──────────────────────────────────────────────────────

func (m model) tabContent(width int) []string {
	switch m.activeTab {
	case 0:
		return m.homeLines(width)
	case 1:
		return m.resultsLines(width)
	case 2:
		return m.duplicatesLines(width)
	default:
		return m.helpLines()
	}
}

// ── Tab 1: Home ─────────────────────────────────────────────────────────────

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
			lines = append(lines, "  "+accentStyle.Render("Go to: ")+valueStyle.Render(m.pathInput)+accentStyle.Render("█"))
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
			switch name {
			case ".":
				styled = accentStyle.Render(displayName)
			case "..":
				styled = mutedStyle.Render(displayName)
			default:
				styled = accentStyle.Render(displayName)
			}
		} else {
			switch name {
			case ".":
				styled = secondaryStyle.Render(displayName)
			case "..":
				styled = mutedStyle.Render(displayName)
				default:
					styled = valueStyle.Render(displayName)
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

	lines = append(lines, m.renderActionMenu(width)...)

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
	}

	var lines []string

	// Box top with title
	// Total line: indent(2) + ┌(1) + titlePart + dashes + ┐(1) = width
	// dashes = width - 4 - len(titlePart)
	titlePart := "─ Actions "
	remaining := width - 4 - lipgloss.Width(titlePart)
	if remaining < 0 {
		remaining = 0
	}
	topBorder := "  " + borderStyle.Render("┌"+titlePart+strings.Repeat("─", remaining)+"┐")
	lines = append(lines, topBorder)

	// Action items
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

		// Watch mode indicator
		extra := ""
		if i == 4 && m.watching {
			extra = " " + successStyle.Render("(active)")
		}

		content := indicator + labelText + extra
		// Right-align shortcut
		contentW := lipgloss.Width(content)
		shortcutW := lipgloss.Width(shortcutText)
		padding := width - 6 - contentW - shortcutW
		if padding < 1 {
			padding = 1
		}
		line := "  " + borderStyle.Render("│") + " " + content + strings.Repeat(" ", padding) + shortcutText + " " + borderStyle.Render("│")
		lines = append(lines, line)
	}

	// Box bottom: indent(2) + └(1) + dashes + ┘(1) = width → dashes = width - 4
	bottomRuler := width - 4
	if bottomRuler < 0 {
		bottomRuler = 0
	}
	bottomBorder := "  " + borderStyle.Render("└"+strings.Repeat("─", bottomRuler)+"┘")
	lines = append(lines, bottomBorder)

	return lines
}

// ── Tab 2: Results ──────────────────────────────────────────────────────────

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

	// Header
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

	return lines
}

// ── Tab 3: Duplicates ───────────────────────────────────────────────────────

func (m model) duplicatesLines(width int) []string {
	var lines []string

	// Scan action
	lines = append(lines,
		"  "+mutedStyle.Render("[d] Scan for duplicates (from Home tab or press d here)"),
		"",
	)

	if m.data.DedupScan == nil {
		lines = append(lines,
			"  "+mutedStyle.Render("No duplicate scan results available."),
			"  "+mutedStyle.Render("Press 'd' on the Home tab to scan."),
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
		lines = append(lines, "  "+successStyle.Render("No duplicates found. Your files are clean!"))
		return lines
	}

	for i, g := range s.DuplicateGroups {
		header := fmt.Sprintf("  Group %d (%s, %d copies):",
			i+1, dedup.FormatSize(g.Size), len(g.Files))
		lines = append(lines, secondaryStyle.Render(header))

		for _, f := range g.Files {
			display := f
			avail := width - 6
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

// ── Tab 4: Help ─────────────────────────────────────────────────────────────

func (m model) helpLines() []string {
	return []string{
		"  " + secondaryStyle.Render("Global shortcuts:"),
		"    " + mutedStyle.Render("1, 2, 3, 4  ") + valueStyle.Render("Switch tabs"),
		"    " + mutedStyle.Render("Tab         ") + valueStyle.Render("Next tab"),
		"    " + mutedStyle.Render("Shift+Tab   ") + valueStyle.Render("Previous tab"),
		"    " + mutedStyle.Render("q / Esc     ") + valueStyle.Render("Quit"),
		"    " + mutedStyle.Render("Ctrl+C      ") + valueStyle.Render("Force quit"),
		"",
		"  " + secondaryStyle.Render("Home tab:"),
		"    " + mutedStyle.Render("↑/↓ or j/k  ") + valueStyle.Render("Navigate action menu"),
		"    " + mutedStyle.Render("Enter       ") + valueStyle.Render("Execute selected action"),
		"    " + mutedStyle.Render("o           ") + valueStyle.Render("Organize files"),
		"    " + mutedStyle.Render("p           ") + valueStyle.Render("Preview (dry-run)"),
		"    " + mutedStyle.Render("d           ") + valueStyle.Render("Scan for duplicates"),
		"    " + mutedStyle.Render("u           ") + valueStyle.Render("Undo last operation"),
		"    " + mutedStyle.Render("w           ") + valueStyle.Render("Toggle watch mode"),
		"    " + mutedStyle.Render("e           ") + valueStyle.Render("Browse and select source directory"),
		"",
		"  " + secondaryStyle.Render("Folder browser:"),
		"    " + mutedStyle.Render("j/k or ↑/↓  ") + valueStyle.Render("Navigate directories"),
		"    " + mutedStyle.Render("Enter on .  ") + valueStyle.Render("Select current folder as target"),
		"    " + mutedStyle.Render("Enter on dir") + valueStyle.Render("Open that directory"),
		"    " + mutedStyle.Render("Enter [X:]  ") + valueStyle.Render("Navigate to a drive (Windows)"),
		"    " + mutedStyle.Render("Backspace   ") + valueStyle.Render("Go to parent directory"),
		"    " + mutedStyle.Render("g           ") + valueStyle.Render("Type or paste a path to navigate directly"),
		"    " + mutedStyle.Render("s           ") + valueStyle.Render("Select current directory"),
		"    " + mutedStyle.Render("Home/End    ") + valueStyle.Render("Jump to first/last entry"),
		"    " + mutedStyle.Render("Esc         ") + valueStyle.Render("Cancel"),
		"",
		"  " + secondaryStyle.Render("Results / Duplicates tabs:"),
		"    " + mutedStyle.Render("j / ↓       ") + valueStyle.Render("Scroll down"),
		"    " + mutedStyle.Render("k / ↑       ") + valueStyle.Render("Scroll up"),
		"",
		"  " + secondaryStyle.Render("Confirmation dialog:"),
		"    " + mutedStyle.Render("y           ") + valueStyle.Render("Confirm"),
		"    " + mutedStyle.Render("n / Esc     ") + valueStyle.Render("Cancel"),
	}
}

// ── Confirmation Dialog ─────────────────────────────────────────────────────

func (m model) confirmLines(width int) []string {
	var lines []string

	// Add some vertical centering
	for i := 0; i < 3; i++ {
		lines = append(lines, "")
	}

	// Box
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

	// Title line
	titleContent := "  Confirm"
	titleLine := pad + borderStyle.Render("│") + titleStyle.Render(titleContent) + strings.Repeat(" ", boxWidth-2-lipgloss.Width(titleContent)) + borderStyle.Render("│")
	lines = append(lines, titleLine)

	// Empty line
	lines = append(lines, pad+borderStyle.Render("│")+strings.Repeat(" ", boxWidth-2)+borderStyle.Render("│"))

	// Message
	msgContent := "  " + m.confirmMsg
	if lipgloss.Width(msgContent) > boxWidth-2 {
		msgContent = truncateMiddle(msgContent, boxWidth-2)
	}
	msgPad := boxWidth - 2 - lipgloss.Width(msgContent)
	if msgPad < 0 {
		msgPad = 0
	}
	lines = append(lines, pad+borderStyle.Render("│")+valueStyle.Render(msgContent)+strings.Repeat(" ", msgPad)+borderStyle.Render("│"))

	// Empty line
	lines = append(lines, pad+borderStyle.Render("│")+strings.Repeat(" ", boxWidth-2)+borderStyle.Render("│"))

	// Buttons
	buttons := "  " + successStyle.Render("[y] Confirm") + "   " + errorStyle.Render("[n] Cancel")
	btnPad := boxWidth - 2 - lipgloss.Width(buttons)
	if btnPad < 0 {
		btnPad = 0
	}
	lines = append(lines, pad+borderStyle.Render("│")+buttons+strings.Repeat(" ", btnPad)+borderStyle.Render("│"))

	// Bottom
	lines = append(lines, pad+borderStyle.Render("└"+innerRuler+"┘"))

	return lines
}

// ── Helpers ─────────────────────────────────────────────────────────────────

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

	// Find max source filename length
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
