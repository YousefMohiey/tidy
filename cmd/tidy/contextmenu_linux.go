//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// registerDirectoryContextMenu installs a Nautilus script on Linux.
func registerDirectoryContextMenu(exePath string) error {
	// Create Nautilus scripts directory
	scriptsDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "nautilus", "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return fmt.Errorf("failed to create scripts directory: %w", err)
	}

	// Create the script
	scriptPath := filepath.Join(scriptsDir, "Organize with tidy")
	script := fmt.Sprintf(`#!/bin/bash
# Organize directory with tidy

# Get the selected directory
if [ -n "$NAUTILUS_SCRIPT_SELECTED_FILE_PATHS" ]; then
    DIR="$NAUTILUS_SCRIPT_SELECTED_FILE_PATHS"
else
    DIR="$(pwd)"
fi

# Run tidy
"%s" organize "$DIR"

# Show notification if notify-send is available
if command -v notify-send &> /dev/null; then
    notify-send "tidy" "Organized $DIR"
fi
`, exePath)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write script: %w", err)
	}

	// Also try to install Thunar custom action (for XFCE)
	thunarDir := filepath.Join(os.Getenv("HOME"), ".config", "Thunar")
	if err := os.MkdirAll(thunarDir, 0755); err == nil {
		ucaPath := filepath.Join(thunarDir, "uca.xml")
		ucaContent := fmt.Sprintf(`<?xml encoding="UTF-8" version="1.0"?>
<actions>
<action>
	<icon>folder-remote</icon>
	<name>Organize with tidy</name>
	<unique-id>tidy-organize-%d</unique-id>
	<command>%s organize "%%f"</command>
	<description>Organize files in this directory</description>
	<patterns>*</patterns>
	<directories/>
</action>
</actions>`, os.Getpid(), exePath)

		// Only write if file doesn't exist (don't overwrite existing Thunar config)
		if _, err := os.Stat(ucaPath); os.IsNotExist(err) {
			os.WriteFile(ucaPath, []byte(ucaContent), 0644)
		}
	}

	return nil
}

// unregisterDirectoryContextMenu removes the Nautilus script.
func unregisterDirectoryContextMenu() error {
	scriptPath := filepath.Join(os.Getenv("HOME"), ".local", "share", "nautilus", "scripts", "Organize with tidy")
	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove script: %w", err)
	}

	return nil
}
