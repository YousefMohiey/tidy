<div align="center">

# tidy

**Smart file organizer for your terminal**

*Organize files by content — not just extension. Preview before moving. Undo anything.*

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20Windows%20%7C%20macOS-lightgrey?style=flat-square)](#installation)
[![Release](https://img.shields.io/github/v/release/YousefMohiey/tidy?style=flat-square)](https://github.com/YousefMohiey/tidy/releases)

[Features](#features) • [Installation](#installation) • [Usage](#usage) • [Configuration](#configuration) • [Architecture](#architecture)

</div>

---

## Features

| | |
|---|---|
| **Content-aware sorting** | Reads file magic bytes, not just extensions. A `.dat` that's actually a PNG goes to `Images/` |
| **Interactive TUI dashboard** | Full control center — organize, preview, dedup, undo, and watch from one screen |
| **Folder browser** | Navigate your filesystem visually to pick directories |
| **Preview mode** | See exactly what will move before touching anything |
| **Full undo** | Every operation is journaled. One key to reverse it all |
| **Duplicate detection** | SHA256 content hashing with size-first optimization |
| **Watch mode** | Auto-organize new files as they appear in a directory |
| **11 file categories** | Images, Documents, Videos, Audio, Archives, Code, Fonts, Executables, Disk Images, Ebooks, 3D Models |
| **300+ file types** | Covers everything from `.jpg` to `.heic`, `.stl` to `.epub` |
| **Cross-platform** | Linux, Windows, macOS — single binary, no dependencies |
| **Conflict resolution** | Smart renaming (`photo.jpg` → `photo_1.jpg`) when names collide |
| **YAML config** | Customize rules, add your own categories |

## Installation

### Download a release

Grab the latest binary from [Releases](https://github.com/YousefMohiey/tidy/releases):

| Platform | File |
|---|---|
| Linux (x86_64) | `tidy-linux-amd64` |
| Windows (x86_64) | `tidy-windows-amd64.exe` |
| macOS (Intel) | `tidy-darwin-amd64` |
| macOS (Apple Silicon) | `tidy-darwin-arm64` |

```bash
# Linux
chmod +x tidy-linux-amd64
sudo mv tidy-linux-amd64 /usr/local/bin/tidy

# macOS (pick your architecture)
chmod +x tidy-darwin-arm64
sudo mv tidy-darwin-arm64 /usr/local/bin/tidy

# Windows
# Rename tidy-windows-amd64.exe to tidy.exe and put it in your PATH
```

### Build from source

Requires [Go 1.22+](https://go.dev/dl/).

```bash
git clone https://github.com/YousefMohiey/tidy.git
cd tidy
go build -o tidy ./cmd/tidy/
```

### Quick install (Linux)

```bash
git clone https://github.com/YousefMohiey/tidy.git && cd tidy
./install.sh
```

## Usage

### Interactive dashboard (recommended)

```bash
tidy
```

Just run `tidy` with no arguments. The interactive dashboard opens:

```
╭─────────────────────────────────────────────────╮
│  tidy dashboard              [1] [2] [3] [4]    │
├─────────────────────────────────────────────────┤
│                                                 │
│  Directory: /home/user/Downloads    [e]: browse │
│  Last organized: 2026-05-24 12:30               │
│  Operations: 8                                  │
│                                                 │
│  ┌─ Actions ──────────────────────────────────┐ │
│  │ > Organize files              [o]          │ │
│  │   Preview (dry-run)           [p]          │ │
│  │   Scan for duplicates         [d]          │ │
│  │   Undo last operation         [u]          │ │
│  │   Toggle watch mode           [w]          │ │
│  └────────────────────────────────────────────┘ │
│                                                 │
│  Status: Ready                                  │
╰─────────────────────────────────────────────────╯
  1-4: tabs  enter: select  e: browse dir  q: quit
```

**Keyboard shortcuts:**

| Key | Action |
|---|---|
| `e` | Browse and select directory |
| `o` | Organize files |
| `p` | Preview (dry-run) — see what would happen |
| `d` | Scan for duplicates |
| `u` | Undo last operation (with confirmation) |
| `w` | Toggle watch mode |
| `1-4` | Switch tabs |
| `j/k` | Navigate / scroll |
| `q` | Quit |

### Command line

```bash
# Organize a directory
tidy organize ~/Downloads

# Preview without moving anything
tidy organize ~/Downloads --dry-run

# Auto-organize new files in real-time
tidy watch ~/Downloads

# Undo the last operation
tidy undo

# Find duplicate files
tidy dedup ~/Downloads

# Find duplicates across multiple directories
tidy dedup ~/Downloads ~/Documents ~/Pictures

# Output duplicates as JSON
tidy dedup ~/Downloads --json

# Show last operation details
tidy status

# Use a custom config
tidy organize ~/Downloads --config my-rules.yaml
```

## Configuration

`tidy` works out of the box with sensible defaults. To customize, create a `config.yaml`:

```yaml
rules:
  - name: Images
    extensions:
      - jpg
      - png
      - gif
      - webp
      - heic
    magic_bytes:
      - image/jpeg
      - image/png
    destination: Images
    patterns: []

  - name: My Custom Category
    extensions:
      - custom
      - special
    magic_bytes: []
    destination: Custom
    patterns:
      - "*.backup"
```

Then use it:

```bash
tidy organize ~/Downloads --config config.yaml
```

### Default categories

| Category | Example extensions |
|---|---|
| **Images** | jpg, png, gif, svg, webp, heic, avif, psd, raw, cr2, nef |
| **Documents** | pdf, doc, docx, txt, md, xlsx, pptx, tex, pages, keynote |
| **Videos** | mp4, avi, mkv, mov, webm, m4v, mpg, 3gp, vob |
| **Audio** | mp3, wav, flac, aac, ogg, aiff, opus, midi |
| **Archives** | zip, tar, gz, 7z, rar, bz2, xz, zst, cab, jar, dmg |
| **Code** | py, js, ts, go, rs, c, cpp, java, rb, php, sh, sql, lua, swift, kt, and 50+ more |
| **Fonts** | ttf, otf, woff, woff2, eot |
| **Executables** | exe, msi, deb, rpm, apk, appimage |
| **Disk Images** | iso, img, vmdk, vdi, vhd, qcow2 |
| **Ebooks** | epub, mobi, azw, djvu, fb2 |
| **3D Models** | stl, obj, fbx, blend, gltf, glb, step |

## Architecture

```
tidy/
├── cmd/tidy/main.go              # CLI entry point (cobra)
├── internal/
│   ├── config/config.go          # YAML config + defaults
│   ├── rules/rules.go            # 3-phase matching: ext → MIME → glob
│   ├── detector/detector.go      # Magic bytes via mimetype lib
│   ├── organizer/
│   │   ├── organizer.go          # File moves + conflict resolution
│   │   └── journal.go            # JSON journal + undo
│   ├── watcher/watcher.go        # fsnotify + debounce (500ms)
│   ├── dedup/dedup.go            # SHA256 dedup (size-first optimization)
│   ├── paths/paths.go            # Platform-specific data directories
│   └── tui/dashboard.go          # Bubble Tea interactive dashboard
├── config.yaml                   # Default rules
└── build.sh                      # Cross-platform build script
```

### How detection works

```
File in Downloads/
    │
    ▼
Detector reads magic bytes → "image/jpeg"
    │
    ▼
Rule Engine matches (3 phases):
  1. Extension match (fastest)
  2. MIME type match
  3. Glob pattern match
    │
    ▼
Organizer creates Images/ dir
    │
    ▼
Conflict check → Images/photo.jpg
    │
    ▼
os.Rename(source, destination)
    │
    ▼
Journal records {source, destination, timestamp}
```

## Data locations

| Platform | Path |
|---|---|
| Linux | `~/.local/share/tidy/journal.json` |
| Windows | `%LOCALAPPDATA%\tidy\journal.json` |
| macOS | `~/Library/Application Support/tidy/journal.json` |

## Building

```bash
# Build for current platform
go build -o tidy ./cmd/tidy/

# Build all platforms
./build.sh

# Output in dist/
#   tidy-linux-amd64        (4.5 MB)
#   tidy-windows-amd64.exe  (4.8 MB)
#   tidy-darwin-amd64       (4.6 MB, Intel)
#   tidy-darwin-arm64       (4.3 MB, Apple Silicon)
```

## License

[MIT](LICENSE)
