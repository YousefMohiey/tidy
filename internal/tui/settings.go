package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) toggleContextMenu() (tea.Model, tea.Cmd) {
	if runtime.GOOS != "windows" {
		m.status = "Context menu is only available on Windows"
		m.statusStyle = errorStyle
		return m, nil
	}

	if m.contextMenuInstalled() {
		m.confirming = true
		m.confirmAction = "remove-context-menu"
		m.confirmMsg = "Remove 'Organize with tidy' from right-click menu?"
		return m, nil
	}

	m.confirming = true
	m.confirmAction = "install-context-menu"
	m.confirmMsg = "Add 'Organize with tidy' to right-click menu?"
	return m, nil
}

func (m model) contextMenuInstalled() bool {
	return m.contextMenuState
}

func (m model) installContextMenu() (tea.Model, tea.Cmd) {
	if runtime.GOOS != "windows" {
		return m, nil
	}
	exe, err := os.Executable()
	if err != nil {
		m.status = "Error: " + err.Error()
		m.statusStyle = errorStyle
		return m, nil
	}
	exe = filepath.Clean(exe)
	iconPath := filepath.Join(filepath.Dir(exe), "tidy.ico")

	_ = registryWriteString(`Software\Classes\Directory\Background\shell\tidy`, "", "Organize with tidy")
	_ = registryWriteString(`Software\Classes\Directory\Background\shell\tidy`, "Icon", iconPath)
	_ = registryWriteString(`Software\Classes\Directory\Background\shell\tidy\command`, "", `cmd.exe /k ""`+exe+`"" organize ""%V""`)

	_ = registryWriteString(`Software\Classes\Directory\shell\tidy`, "", "Organize with tidy")
	_ = registryWriteString(`Software\Classes\Directory\shell\tidy`, "Icon", iconPath)
	_ = registryWriteString(`Software\Classes\Directory\shell\tidy\command`, "", `cmd.exe /k ""`+exe+`"" organize ""%1""`)

	m.contextMenuState = true
	m.status = "Context menu installed"
	m.statusStyle = successStyle
	return m, nil
}

func (m model) removeContextMenu() (tea.Model, tea.Cmd) {
	if runtime.GOOS != "windows" {
		return m, nil
	}
	_ = registryDeleteKey(`Software\Classes\Directory\Background\shell\tidy`)
	_ = registryDeleteKey(`Software\Classes\Directory\shell\tidy`)
	m.contextMenuState = false
	m.status = "Context menu removed"
	m.statusStyle = successStyle
	return m, nil
}

func registryWriteString(path, name, value string) error {
	args := []string{"add", "HKCU\\" + path, "/f"}
	if name != "" {
		args = append(args, "/v", name)
	} else {
		args = append(args, "/ve")
	}
	args = append(args, "/t", "REG_SZ", "/d", value)
	cmd := exec.Command("reg", args...)
	return cmd.Run()
}

func registryDeleteKey(path string) error {
	cmd := exec.Command("reg", "delete", "HKCU\\"+path, "/f")
	return cmd.Run()
}
