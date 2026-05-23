package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

func DataDir() string {
	switch runtime.GOOS {
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return filepath.Join(local, "tidy")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "AppData", "Local", "tidy")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "tidy")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share", "tidy")
	}
}

func JournalPath() string {
	return filepath.Join(DataDir(), "journal.json")
}
