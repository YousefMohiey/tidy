package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level YAML configuration for tidy.
type Config struct {
	Rules []Rule `yaml:"rules"`
}

// Rule defines a single file-organization rule.
// A file matches a rule if it satisfies ANY of the rule's criteria
// (extensions, magic_bytes, or patterns).
type Rule struct {
	Name        string   `yaml:"name"`
	Extensions  []string `yaml:"extensions"`
	MagicBytes  []string `yaml:"magic_bytes"` // MIME types to match
	Destination string   `yaml:"destination"` // relative folder name e.g. "Images"
	Patterns    []string `yaml:"patterns"`    // filename glob patterns
}

// Load reads a YAML configuration file from disk and parses it into a Config.
// Returns an error with the file path context if reading or parsing fails.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config: failed to read file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: failed to parse file %q: %w", path, err)
	}

	if len(cfg.Rules) == 0 {
		return nil, fmt.Errorf("config: file %q contains no rules", path)
	}

	return &cfg, nil
}

// Default returns a hardcoded Config with sensible default rules.
// These defaults mirror the contents of config.yaml.
func Default() *Config {
	return &Config{
		Rules: []Rule{
			{
				Name:       "Images",
				Extensions: []string{"jpg", "jpeg", "png", "gif", "bmp", "svg", "webp", "ico", "tiff", "tif"},
				MagicBytes: []string{
					"image/jpeg", "image/png", "image/gif", "image/bmp",
					"image/svg+xml", "image/webp", "image/x-icon", "image/tiff",
				},
				Destination: "Images",
				Patterns:    []string{},
			},
			{
				Name:       "Documents",
				Extensions: []string{"pdf", "doc", "docx", "txt", "md", "rtf", "odt", "xls", "xlsx", "ppt", "pptx", "csv"},
				MagicBytes: []string{
					"application/pdf",
					"application/msword",
					"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
					"text/plain",
					"text/markdown",
					"application/rtf",
					"text/rtf",
					"application/vnd.oasis.opendocument.text",
					"application/vnd.ms-excel",
					"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
					"application/vnd.ms-powerpoint",
					"application/vnd.openxmlformats-officedocument.presentationml.presentation",
					"text/csv",
				},
				Destination: "Documents",
				Patterns:    []string{"*.txt", "*.md", "*.csv"},
			},
			{
				Name:       "Videos",
				Extensions: []string{"mp4", "avi", "mkv", "mov", "wmv", "flv", "webm"},
				MagicBytes: []string{
					"video/mp4", "video/x-msvideo", "video/x-matroska",
					"video/quicktime", "video/x-ms-wmv", "video/x-flv", "video/webm",
				},
				Destination: "Videos",
				Patterns:    []string{},
			},
			{
				Name:       "Audio",
				Extensions: []string{"mp3", "wav", "flac", "aac", "ogg", "wma", "m4a"},
				MagicBytes: []string{
					"audio/mpeg", "audio/wav", "audio/x-wav", "audio/flac",
					"audio/aac", "audio/ogg", "audio/x-ms-wma", "audio/mp4",
				},
				Destination: "Audio",
				Patterns:    []string{},
			},
			{
				Name:       "Archives",
				Extensions: []string{"zip", "tar", "gz", "7z", "rar", "bz2", "xz", "zst"},
				MagicBytes: []string{
					"application/zip", "application/x-tar", "application/gzip",
					"application/x-7z-compressed", "application/vnd.rar",
					"application/x-bzip2", "application/x-xz", "application/zstd",
				},
				Destination: "Archives",
				Patterns:    []string{"*.tar.gz", "*.tar.bz2", "*.tar.xz", "*.tar.zst"},
			},
			{
				Name:       "Code",
				Extensions: []string{
					"go", "py", "js", "ts", "jsx", "tsx", "rs", "c", "cpp", "h", "hpp",
					"java", "rb", "php", "sh", "css", "html", "json", "yaml", "yml",
					"toml", "xml",
				},
				MagicBytes: []string{
					"text/x-go", "text/x-python", "application/javascript",
					"application/typescript", "text/x-rust", "text/x-c",
					"text/x-c++", "text/x-java", "application/x-ruby",
					"text/x-php", "application/x-sh", "text/css",
					"text/html", "application/json", "text/yaml",
					"application/toml", "text/xml", "application/xml",
				},
				Destination: "Code",
				Patterns:    []string{},
			},
		},
	}
}
