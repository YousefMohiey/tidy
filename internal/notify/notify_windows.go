//go:build windows

package notify

import (
	"fmt"
	"github.com/go-toast/toast"
)

// OrganizeComplete shows a Windows toast notification when organize completes.
func OrganizeComplete(filesMoved int, sourceDir string) error {
	notification := toast.Notification{
		AppID:   "tidy",
		Title:   "tidy",
		Message: fmt.Sprintf("Organized %d files in %s", filesMoved, sourceDir),
	}

	return notification.Push()
}
