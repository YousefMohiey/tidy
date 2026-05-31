package rules

import (
	"testing"

	"github.com/YousefMohiey/tidy/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		Rules: []config.Rule{
			{
				Name:        "Images",
				Extensions:  []string{"jpg", "jpeg", "png", "gif"},
				MagicBytes:  []string{"image/jpeg", "image/png"},
				Destination: "Images",
				Patterns:    []string{},
			},
			{
				Name:        "Documents",
				Extensions:  []string{"pdf", "docx"},
				MagicBytes:  []string{"application/pdf"},
				Destination: "Documents",
				Patterns:    []string{"*.doc"},
			},
			{
				Name:        "ConfigFiles",
				Extensions:  []string{"yaml", "yml", "gitignore"},
				MagicBytes:  []string{},
				Destination: "ConfigFiles",
				Patterns:    []string{".*"},
			},
		},
	}
}

func TestMatch(t *testing.T) {
	engine := NewEngine(testConfig())

	tests := []struct {
		name       string
		filename   string
		mimeType   string
		wantNil    bool
		wantCat    string
		wantDest   string
	}{
		{
			name:     "extension match jpg",
			filename: "photo.jpg",
			mimeType: "",
			wantNil:  false,
			wantCat:  "Images",
			wantDest: "Images",
		},
		{
			name:     "extension match case insensitive JPG",
			filename: "photo.JPG",
			mimeType: "",
			wantNil:  false,
			wantCat:  "Images",
			wantDest: "Images",
		},
		{
			name:     "extension match mixed case Png",
			filename: "image.Png",
			mimeType: "",
			wantNil:  false,
			wantCat:  "Images",
			wantDest: "Images",
		},
		{
			name:     "MIME fallback when extension unknown",
			filename: "unknown.xyz",
			mimeType: "image/jpeg",
			wantNil:  false,
			wantCat:  "Images",
			wantDest: "Images",
		},
		{
			name:     "MIME fallback image/png",
			filename: "data.bin",
			mimeType: "image/png",
			wantNil:  false,
			wantCat:  "Images",
			wantDest: "Images",
		},
		{
			name:     "glob pattern match *.doc",
			filename: "report.doc",
			mimeType: "",
			wantNil:  false,
			wantCat:  "Documents",
			wantDest: "Documents",
		},
		{
			name:     "no match returns nil",
			filename: "mystery",
			mimeType: "application/octet-stream",
			wantNil:  true,
		},
		{
			name:     "no match empty mime and unknown ext",
			filename: "file.abc",
			mimeType: "",
			wantNil:  true,
		},
		{
			name:     "extension wins over MIME",
			filename: "file.jpg",
			mimeType: "text/plain",
			wantNil:  false,
			wantCat:  "Images",
			wantDest: "Images",
		},
		{
			name:     "extension wins over MIME conflicting pdf",
			filename: "file.pdf",
			mimeType: "image/jpeg",
			wantNil:  false,
			wantCat:  "Documents",
			wantDest: "Documents",
		},
		{
			name:     "MIME wins over glob pattern",
			filename: "report.doc",
			mimeType: "application/pdf",
			wantNil:  false,
			wantCat:  "Documents",
			wantDest: "Documents",
		},
		{
			name:     "dotfile .gitignore matches extension",
			filename: ".gitignore",
			mimeType: "",
			wantNil:  false,
			wantCat:  "ConfigFiles",
			wantDest: "ConfigFiles",
		},
		{
			name:     "dotfile without ext matches glob pattern",
			filename: ".env",
			mimeType: "",
			wantNil:  false,
			wantCat:  "ConfigFiles",
			wantDest: "ConfigFiles",
		},
		{
			name:     "extension with leading dot in rule",
			filename: "photo.jpeg",
			mimeType: "",
			wantNil:  false,
			wantCat:  "Images",
			wantDest: "Images",
		},
		{
			name:     "empty filename and empty mime",
			filename: "",
			mimeType: "",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.Match(tt.filename, tt.mimeType)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected match, got nil")
			}
			if got.Category != tt.wantCat {
				t.Errorf("Category = %q, want %q", got.Category, tt.wantCat)
			}
			if got.Destination != tt.wantDest {
				t.Errorf("Destination = %q, want %q", got.Destination, tt.wantDest)
			}
		})
	}
}

func TestNewEngine(t *testing.T) {
	cfg := testConfig()
	e := NewEngine(cfg)
	if e == nil {
		t.Fatal("NewEngine returned nil")
	}
}

func TestExtFromName(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"photo.jpg", ".jpg"},
		{"archive.tar.gz", ".gz"},
		{".gitignore", ".gitignore"},
		{"noext", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := extFromName(tt.filename)
			if got != tt.want {
				t.Errorf("extFromName(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}
