//go:build !windows

package notify

// OrganizeComplete is a no-op on non-Windows platforms.
// Linux uses notify-send in the Nautilus script, macOS uses native notifications.
func OrganizeComplete(filesMoved int, sourceDir string) error {
	return nil
}
