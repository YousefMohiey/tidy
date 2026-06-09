package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	for i, rule := range cfg.Rules {
		if err := validateDestination(rule.Destination); err != nil {
			return nil, fmt.Errorf("config: rule %d (%q): %w", i, rule.Name, err)
		}
		if rule.Name == "" {
			return nil, fmt.Errorf("config: rule %d has empty name", i)
		}
	}

	return &cfg, nil
}

func validateDestination(dest string) error {
	if dest == "" {
		return fmt.Errorf("destination must not be empty")
	}
	if filepath.IsAbs(dest) {
		return fmt.Errorf("destination %q must be relative, not absolute", dest)
	}
	cleaned := filepath.Clean(dest)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("destination %q contains path traversal", dest)
	}
	if strings.Contains(cleaned, string(filepath.Separator)+".."+string(filepath.Separator)) {
		return fmt.Errorf("destination %q contains path traversal", dest)
	}
	return nil
}

// Default returns a hardcoded Config with sensible default rules.
// These defaults mirror the contents of config.yaml.
func Default() *Config {
	return &Config{
		Rules: []Rule{
			{
				Name: "Images",
				Extensions: []string{
					"jpg", "jpeg", "png", "gif", "bmp", "svg", "webp", "ico", "tiff", "tif",
					"heic", "heif", "avif", "jxl", "raw", "cr2", "nef", "arw", "dng",
					"psd", "ai", "eps", "xcf", "tga", "indd",
				},
				MagicBytes: []string{
					"image/jpeg", "image/png", "image/gif", "image/bmp",
					"image/svg+xml", "image/webp", "image/x-icon", "image/tiff",
					"image/heic", "image/heif", "image/avif", "image/jxl",
					"image/x-canon-cr2", "image/x-nikon-nef", "image/x-sony-arw",
					"image/x-adobe-dng", "image/vnd.adobe.photoshop",
					"image/x-xcf", "image/x-tga",
				},
				Destination: "Images",
				Patterns:    []string{},
			},
			{
				Name: "Documents",
				Extensions: []string{
					"pdf", "doc", "docx", "txt", "md", "rtf", "odt", "xls", "xlsx", "ppt", "pptx", "csv",
					"tex", "pages", "numbers", "keynote", "wps", "vsd", "pub", "msg",
					"ods", "odp", "sxc", "sxi", "sxw",
				},
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
					"application/x-tex",
					"application/vnd.apple.pages",
					"application/vnd.apple.numbers",
					"application/vnd.apple.keynote",
					"application/vnd.oasis.opendocument.spreadsheet",
					"application/vnd.oasis.opendocument.presentation",
				},
				Destination: "Documents",
				Patterns:    []string{"*.txt", "*.md", "*.csv", "*.tex"},
			},
			{
				Name: "Videos",
				Extensions: []string{
					"mp4", "avi", "mkv", "mov", "wmv", "flv", "webm",
					"m4v", "mpg", "mpeg", "3gp", "ts", "vob", "ogv", "rm", "asf", "divx",
				},
				MagicBytes: []string{
					"video/mp4", "video/x-msvideo", "video/x-matroska",
					"video/quicktime", "video/x-ms-wmv", "video/x-flv", "video/webm",
					"video/x-m4v", "video/mpeg", "video/3gpp",
					"video/mp2t", "video/x-ms-vob", "video/ogg",
					"application/vnd.rn-realmedia",
				},
				Destination: "Videos",
				Patterns:    []string{},
			},
			{
				Name: "Audio",
				Extensions: []string{
					"mp3", "wav", "flac", "aac", "ogg", "wma", "m4a",
					"aiff", "ape", "opus", "mid", "midi", "amr", "weba",
				},
				MagicBytes: []string{
					"audio/mpeg", "audio/wav", "audio/x-wav", "audio/flac",
					"audio/aac", "audio/ogg", "audio/x-ms-wma", "audio/mp4",
					"audio/aiff", "audio/x-ape", "audio/opus",
					"audio/midi", "audio/amr", "audio/webm",
				},
				Destination: "Audio",
				Patterns:    []string{},
			},
			{
				Name: "Archives",
				Extensions: []string{
					"zip", "tar", "gz", "7z", "rar", "bz2", "xz", "zst",
					"cab", "jar", "war", "ear", "cpio", "lzh", "arj", "lz", "lzma", "lz4",
					"sz", "z", "dmg", "iso",
				},
				MagicBytes: []string{
					"application/zip", "application/x-tar", "application/gzip",
					"application/x-7z-compressed", "application/vnd.rar",
					"application/x-bzip2", "application/x-xz", "application/zstd",
					"application/vnd.ms-cab-compressed",
					"application/java-archive",
					"application/x-cpio",
					"application/x-apple-diskimage",
					"application/x-iso9660-image",
				},
				Destination: "Archives",
				Patterns: []string{
					"*.tar.gz", "*.tar.bz2", "*.tar.xz", "*.tar.zst",
					"*.tar.lz", "*.tar.lzma", "*.tar.lz4",
				},
			},
			{
				Name: "Code",
				Extensions: []string{
					"go", "py", "js", "ts", "jsx", "tsx", "rs", "c", "cpp", "h", "hpp",
					"java", "rb", "php", "sh", "css", "html", "json", "yaml", "yml",
					"toml", "xml",
					"lua", "pl", "r", "swift", "kt", "kts", "scala", "dart", "ex", "exs",
					"erl", "hs", "ml", "clj", "cljs", "lisp", "el", "pas", "vb", "bas",
					"bat", "ps1", "psd1", "psm1", "cmd",
					"sql", "asm", "s", "vim",
					"scss", "less", "sass", "vue", "svelte", "astro", "mdx",
					"graphql", "gql", "proto",
					"tf", "hcl", "ini", "cfg", "conf", "env", "properties", "dotenv",
					"dockerfile", "makefile", "cmake",
					"zig", "nim", "v", "jl", "f90", "f95", "f03",
					"cs", "fs", "fsx", "fsi",
					"m", "mm",
					"groovy", "gradle",
					"awk", "sed",
					"ipynb",
				},
				MagicBytes: []string{
					"text/x-go", "text/x-python", "application/javascript",
					"application/typescript", "text/x-rust", "text/x-c",
					"text/x-c++", "text/x-java", "application/x-ruby",
					"text/x-php", "application/x-sh", "text/css",
					"text/html", "application/json", "text/yaml",
					"application/toml", "text/xml", "application/xml",
					"text/x-lua", "text/x-perl", "text/x-r",
					"text/x-swift", "text/x-kotlin", "text/x-scala",
					"text/x-dart", "text/x-erlang", "text/x-haskell",
					"text/x-ocaml", "text/x-clojure", "text/x-pascal",
					"text/x-sql", "text/x-asm",
					"text/x-scss", "text/x-less",
					"text/x-ini",
				},
				Destination: "Code",
				Patterns:    []string{},
			},
			{
				Name:       "Fonts",
				Extensions: []string{"ttf", "otf", "woff", "woff2", "eot", "fon", "ttc"},
				MagicBytes: []string{
					"font/ttf", "font/otf", "font/woff", "font/woff2",
					"application/x-font-ttf", "application/x-font-otf",
					"application/font-woff", "application/font-woff2",
					"application/vnd.ms-fontobject",
				},
				Destination: "Fonts",
				Patterns:    []string{},
			},
			{
				Name:       "Executables",
				Extensions: []string{"exe", "msi", "deb", "rpm", "apk", "appimage", "flatpak", "snap", "bin", "run", "AppImage"},
				MagicBytes: []string{
					"application/x-executable",
					"application/x-msdos-program",
					"application/x-msi",
					"application/vnd.debian.binary-package",
					"application/x-rpm",
					"application/vnd.android.package-archive",
					"application/x-iso9660-appimage",
				},
				Destination: "Executables",
				Patterns:    []string{},
			},
			{
				Name: "Disk Images",
				Extensions: []string{
					"iso", "img", "vmdk", "vdi", "vhd", "vhdx", "qcow2", "raw",
					"bin", "cue", "nrg", "mdf", "mds",
				},
				MagicBytes: []string{
					"application/x-iso9660-image",
					"application/x-qemu-disk",
					"application/x-vmdk-disk",
					"application/x-virtualbox-vdi",
					"application/x-virtualbox-vhd",
				},
				Destination: "Disk Images",
				Patterns:    []string{},
			},
			{
				Name: "Ebooks",
				Extensions: []string{"epub", "mobi", "azw", "azw3", "djvu", "fb2", "lit", "lrf", "pdb"},
				MagicBytes: []string{
					"application/epub+zip",
					"application/x-mobipocket-ebook",
					"application/vnd.amazon.ebook",
					"image/vnd.djvu",
					"application/x-fictionbook+xml",
				},
				Destination: "Ebooks",
				Patterns:    []string{},
			},
			{
				Name: "3D Models",
				Extensions: []string{
					"stl", "obj", "fbx", "blend", "3ds", "dae", "gltf", "glb",
					"ply", "wrl", "x3d", "iges", "igs", "step", "stp",
				},
				MagicBytes: []string{
					"model/stl", "model/obj", "model/fbx",
					"application/x-blender",
					"model/gltf-binary", "model/gltf+json",
					"model/vrml", "model/x3d",
				},
				Destination: "3D Models",
				Patterns:    []string{},
			},
		},
	}
}
