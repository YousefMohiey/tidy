package tui

import "github.com/charmbracelet/lipgloss"

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
	cursorStyle      = lipgloss.NewStyle().Reverse(true)
	hintStyle        = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
)
