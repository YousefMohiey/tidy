// Package detector provides MIME type detection for files using magic bytes
// with extension-based fallback.
package detector

import (
	"fmt"
	"path/filepath"

	"github.com/gabriel-vasile/mimetype"
)

// Detect returns the MIME type of a file by reading its magic bytes.
// Falls back to extension-based detection if magic bytes are inconclusive.
// Returns "application/octet-stream" if completely unknown.
func Detect(filePath string) (string, error) {
	mime, err := mimetype.DetectFile(filePath)
	if err != nil {
		return "", fmt.Errorf("detect %s: %w", filePath, err)
	}

	result := mime.String()
	if result == "application/octet-stream" {
		if ext := DetectByExtension(filePath); ext != "" {
			return ext, nil
		}
	}

	return result, nil
}

// DetectByExtension returns MIME type based on file extension only.
// Returns empty string if the extension is unknown.
func DetectByExtension(filePath string) string {
	ext := filepath.Ext(filePath)
	if ext == "" {
		return ""
	}

	mime := mimetype.Lookup(ext)
	if mime == nil {
		return ""
	}

	return mime.String()
}
