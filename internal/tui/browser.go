package tui

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleBrowseKey(key string) (tea.Model, tea.Cmd) {
	if m.typingPath {
		return m.handlePathInputKey(key)
	}
	switch key {
	case "g":
		m.typingPath = true
		m.pathInput = m.browsePath
		m.pathCursor = len(m.pathInput)
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
			m.treePreview = nil
			return m, nil
		}
		if name == ".." {
			m.browsePath = filepath.Dir(m.browsePath)
			if m.browsePath == "." {
				m.browsePath = string(os.PathSeparator)
			}
		} else if runtime.GOOS == "windows" && len(name) == 2 && name[1] == ':' {
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
	case "s":
		m.data.SourceDir = m.browsePath
		m.browsingDir = false
		m.treePreview = nil
		return m, nil
	case "esc":
		m.browsingDir = false
		m.treePreview = nil
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
		} else {
			m.status = "Invalid path: " + cleanPath
			m.statusStyle = errorStyle
		}
		m.typingPath = false
		m.pathInput = ""
		m.pathCursor = 0
		return m, nil
	case "esc":
		m.typingPath = false
		m.pathInput = ""
		m.pathCursor = 0
		return m, nil
	case "left":
		if m.pathCursor > 0 {
			m.pathCursor--
		}
		return m, nil
	case "right":
		if m.pathCursor < len(m.pathInput) {
			m.pathCursor++
		}
		return m, nil
	case "home":
		m.pathCursor = 0
		return m, nil
	case "end":
		m.pathCursor = len(m.pathInput)
		return m, nil
	case "backspace":
		if m.pathCursor > 0 {
			m.pathInput = m.pathInput[:m.pathCursor-1] + m.pathInput[m.pathCursor:]
			m.pathCursor--
		}
		return m, nil
	case "ctrl+u":
		m.pathInput = ""
		m.pathCursor = 0
		return m, nil
	default:
		for _, r := range key {
			if r >= 32 && r < 127 {
				m.pathInput = m.pathInput[:m.pathCursor] + string(r) + m.pathInput[m.pathCursor:]
				m.pathCursor++
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
		drives := listWindowsDrives()
		result = append(result, drives...)
	}

	result = append(result, dirs...)
	return result
}
