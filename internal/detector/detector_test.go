package detector

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0644); err != nil {
		t.Fatalf("write test file %s: %v", p, err)
	}
	return p
}

func minimalPNG() []byte {
	var buf []byte
	// PNG signature
	buf = append(buf, 0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A)
	// IHDR chunk: length=13, type=IHDR, 1x1 8-bit RGB, CRC
	buf = append(buf, 0x00, 0x00, 0x00, 0x0D) // length
	buf = append(buf, 0x49, 0x48, 0x44, 0x52) // "IHDR"
	buf = append(buf, 0x00, 0x00, 0x00, 0x01) // width=1
	buf = append(buf, 0x00, 0x00, 0x00, 0x01) // height=1
	buf = append(buf, 0x08)                   // bit depth=8
	buf = append(buf, 0x02)                   // color type=RGB
	buf = append(buf, 0x00, 0x00, 0x00)       // compression, filter, interlace
	buf = append(buf, 0x90, 0x77, 0x53, 0xDE) // CRC
	// IDAT chunk: minimal compressed data (zlib deflate of single row)
	buf = append(buf, 0x00, 0x00, 0x00, 0x0C) // length
	buf = append(buf, 0x49, 0x44, 0x41, 0x54) // "IDAT"
	buf = append(buf, 0x08, 0xD7, 0x63, 0x00, 0x00, 0x00, 0x04, 0x00, 0x01) // compressed data
	buf = append(buf, 0x00, 0x00, 0x00, 0x00) // placeholder CRC
	// IEND chunk
	buf = append(buf, 0x00, 0x00, 0x00, 0x00) // length
	buf = append(buf, 0x49, 0x45, 0x4E, 0x44) // "IEND"
	buf = append(buf, 0xAE, 0x42, 0x60, 0x82) // CRC
	return buf
}

func minimalJPEG() []byte {
	// SOI + APP0 marker header - enough for mimetype library detection
	return []byte{
		0xFF, 0xD8, 0xFF, 0xE0, // SOI + APP0
		0x00, 0x10, // length=16
		0x4A, 0x46, 0x49, 0x46, 0x00, // "JFIF\0"
		0x01, 0x01, // version 1.1
		0x00,       // aspect ratio units
		0x00, 0x01, // X density
		0x00, 0x01, // Y density
		0x00, 0x00, // thumbnail dimensions
	}
}

func TestDetect_PNG(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.png", minimalPNG())

	mime, err := Detect(path)
	if err != nil {
		t.Fatalf("Detect(%s) error: %v", path, err)
	}
	if mime != "image/png" {
		t.Errorf("Detect(%s) = %q, want %q", path, mime, "image/png")
	}
}

func TestDetect_JPEG(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.jpg", minimalJPEG())

	mime, err := Detect(path)
	if err != nil {
		t.Fatalf("Detect(%s) error: %v", path, err)
	}
	if mime != "image/jpeg" {
		t.Errorf("Detect(%s) = %q, want %q", path, mime, "image/jpeg")
	}
}

func TestDetect_UnknownContent(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "noext", []byte{0x00, 0x01, 0x02, 0x03})

	mime, err := Detect(path)
	if err != nil {
		t.Fatalf("Detect(%s) error: %v", path, err)
	}
	if mime != "application/octet-stream" {
		t.Errorf("Detect(%s) = %q, want %q", path, mime, "application/octet-stream")
	}
}

func TestDetect_NonexistentFile(t *testing.T) {
	_, err := Detect(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatal("Detect() on nonexistent file should return error")
	}
}

func TestDetectByExtension_Known(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantMIME string
	}{
		{"png", "foo.png", "image/png"},
		{"jpeg", "foo.jpg", "image/jpeg"},
		{"jpeg_long", "photo.jpeg", "image/jpeg"},
		{"pdf", "doc.pdf", "application/pdf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectByExtension(tt.path)
			if got != tt.wantMIME {
				t.Errorf("DetectByExtension(%q) = %q, want %q", tt.path, got, tt.wantMIME)
			}
		})
	}
}

func TestDetectByExtension_Unknown(t *testing.T) {
	got := DetectByExtension("foo.unknownext123")
	if got != "" {
		t.Errorf("DetectByExtension(%q) = %q, want empty string", "foo.unknownext123", got)
	}
}

func TestDetectByExtension_NoExtension(t *testing.T) {
	got := DetectByExtension("noext")
	if got != "" {
		t.Errorf("DetectByExtension(%q) = %q, want empty string", "noext", got)
	}
}

func TestDetect_FallbackToExtension(t *testing.T) {
	dir := t.TempDir()
	// Write random bytes with a .png extension — magic bytes inconclusive,
	// should fall back to extension-based detection.
	path := writeTestFile(t, dir, "fake.png", []byte{0x00, 0x01, 0x02, 0x03})

	mime, err := Detect(path)
	if err != nil {
		t.Fatalf("Detect(%s) error: %v", path, err)
	}
	if mime != "image/png" {
		t.Errorf("Detect(%s) = %q, want %q (extension fallback)", path, mime, "image/png")
	}
}
