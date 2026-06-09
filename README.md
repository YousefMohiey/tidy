<div align="center">

<img src="installer/tidy-banner.svg?v=4" alt="tidy banner" width="800">

<br/>

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20Windows%20%7C%20macOS-lightgrey?style=flat-square)](#installation)
[![Release](https://img.shields.io/github/v/release/YousefMohiey/tidy?style=flat-square)](https://github.com/YousefMohiey/tidy/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/YousefMohiey/tidy?style=flat-square)](https://goreportcard.com/report/github.com/YousefMohiey/tidy)
[![Downloads](https://img.shields.io/github/downloads/YousefMohiey/tidy/total?style=flat-square)](https://github.com/YousefMohiey/tidy/releases)

**tidy** is a cross-platform file organization tool that sorts your files by their actual content type — not just their extension.  
It features an interactive TUI dashboard, duplicate detection, real-time watch mode, and full undo support.

[Features](#features) • [Installation](#installation) • [Quick Start](#quick-start) • [Usage](#usage) • [Configuration](#configuration) • [Architecture](#architecture) • [Building](#building)

</div>

---

## Features

| | |
|---|---|
| **🧠 Content-aware sorting** | Reads file magic bytes, not just extensions — a `.jpg` renamed to `.png` still goes to Images |
| **🖥️ Interactive TUI dashboard** | Organize, preview, dedup, undo, and watch from one screen. Full keyboard navigation |
| **🔍 Preview mode** | See exactly what will move before touching anything — zero risk |
| **↩️ Full undo** | Journaled with atomic crash-safe saves. Undo any batch operation |
| **📜 Undo history** | Browse and undo from past operations, not just the last one |
| **♻️ Duplicate detection** | SHA256 hashing with speed-optimized persistent cache |
| **✅ Duplicate resolver** | Select which copies to keep or delete with freed-space report |
| **👁️ Watch mode** | Auto-organize new files in real-time using fsnotify |
| **🌳 Tree preview** | Visualize file categorization before organizing |
| **📊 Progress reporting** | Live counts during organize and dedup operations |
| **📁 11 categories** | Images, Documents, Videos, Audio, Archives, Code, Fonts, Executables, Disk Images, Ebooks, 3D Models |
| **📄 300+ file types** | `.jpg` to `.heic`, `.stl` to `.epub` — broad coverage |
| **💻 Cross-platform** | Linux, Windows, macOS — single static binary, no dependencies |
| **🏷️ Smart renaming** | `photo.jpg` → `photo_1.jpg` on name collisions |
| **⚙️ YAML config** | Customize rules and categories via a simple config file |
| **📌 Windows context menu** | Right-click any folder → Organize with tidy |
| **🎨 ANSI colors** | Full color on all terminals including Windows CMD |
| **📂 Folder browser** | Navigate visually or type/paste paths directly (`g` key) |

---

## Installation

### Linux

| Package | Command |
|---|---|
| **Debian / Ubuntu (.deb)** | `wget https://github.com/YousefMohiey/tidy/releases/latest/download/tidy-1.2.0-amd64.deb && sudo dpkg -i tidy-1.2.0-amd64.deb` |
| **Fedora / RHEL (.rpm)** | `wget https://github.com/YousefMohiey/tidy/releases/latest/download/tidy-1.2.0-1.x86_64.rpm && sudo rpm -i tidy-1.2.0-1.x86_64.rpm` |
| **Portable binary** | `wget https://github.com/YousefMohiey/tidy/releases/latest/download/tidy-linux-amd64 -O tidy && chmod +x tidy && sudo mv tidy /usr/local/bin/tidy` |

> 💡 `.deb` installs to `/usr/bin/tidy`, `.rpm` to `/usr/bin/tidy` — both add a desktop entry and icon automatically.

### macOS

```bash
# Intel Mac
curl -L https://github.com/YousefMohiey/tidy/releases/latest/download/tidy-macos-intel -o tidy
chmod +x tidy && sudo mv tidy /usr/local/bin/

# Apple Silicon
curl -L https://github.com/YousefMohiey/tidy/releases/latest/download/tidy-macos-arm64 -o tidy
chmod +x tidy && sudo mv tidy /usr/local/bin/
```

### Windows

Download **`tidy-Setup-x.x.x.exe`** from the [Releases page](https://github.com/YousefMohiey/tidy/releases).

- Installs to `%LOCALAPPDATA%\Programs\tidy`
- Adds to `PATH` automatically
- Creates Start Menu and Desktop shortcuts
- **No admin required** — user-scope only

### Build from source

```bash
git clone https://github.com/YousefMohiey/tidy.git
cd tidy
go build -o tidy ./cmd/tidy/
sudo mv tidy /usr/local/bin/
```

---

## Quick Start

```bash
# Fire up the interactive dashboard
tidy

# Or organize a directory right from the CLI
tidy organize ~/Downloads

# See what would happen without actually moving anything
tidy organize --dry-run ~/Downloads
```

Running `tidy` with no arguments opens the **TUI dashboard** — your central hub for all operations.

---

## Usage

### Interactive dashboard

```
tidy
```

**Keyboard shortcuts:**

| Key | Action |
|---|---|
| `e` | Browse and select directory |
| `o` | Organize files |
| `p` | Preview (dry-run) |
| `d` | Scan for duplicates |
| `u` | Undo last operation (with confirmation) |
| `w` | Toggle watch mode |
| `g` | Type/paste a path directly |
| `s` | Select current directory |
| `← →` | Move cursor in path input |
| `Backspace` | Go to parent directory |
| `1-4` | Switch tabs |
| `j/k` or `↑/↓` | Navigate / scroll |
| `q` | Quit |

### CLI commands

```bash
tidy organize ~/Downloads                # Organize files
tidy organize --dry-run ~/Downloads       # Preview only (safe)
tidy watch ~/Downloads                    # Watch + auto-organize
tidy undo                                 # Undo last operation
tidy dedup ~/Downloads                    # Find duplicates
tidy dedup ~/Downloads --json             # JSON output for scripting
tidy status                               # Last operation summary
tidy organize ~/Downloads --config rules.yaml  # Custom rules
```

---

## Configuration

tidy uses a YAML config file for custom rules and categories.  
Default location: `~/.config/tidy/config.yaml`

```yaml
rules:
  - name: Images
    extensions: [jpg, png, gif, webp, heic]
    magic_bytes: [image/jpeg, image/png]
    destination: Images
    patterns: []

  - name: Custom
    extensions: [custom, special]
    magic_bytes: []
    destination: Custom
    patterns: ["*.backup"]
```

> **Security:** Destinations are validated for path traversal safety — `../../etc` is rejected.

---

## Architecture

```
tidy/
├── cmd/tidy/          # CLI entry point + Windows color support
├── internal/
│   ├── config/        # YAML config loading + path validation
│   ├── rules/         # 3-phase matching engine
│   ├── detector/      # MIME detection via magic bytes
│   ├── organizer/     # File moves + atomic journal/undo
│   ├── watcher/       # fsnotify watcher (async organize)
│   ├── dedup/         # SHA256 dedup + persistent cache
│   ├── paths/         # Platform-specific data directories
│   ├── notify/        # Desktop notifications
│   └── tui/           # 9 Bubble Tea TUI modules
├── installer/         # NSIS, RPM, DEB packaging
├── build.sh           # Cross-platform build script
└── config.yaml        # Default rules
```

---

## Building

```bash
./build.sh
```

Produces all platform binaries and packages:
- Linux: `tidy-linux-amd64`, `.deb`, `.rpm`
- Windows: `tidy-Setup-x.x.x.exe`, `tidy-windows-amd64.exe`
- macOS: `tidy-macos-intel`, `tidy-macos-arm64`

---

## License

[MIT](LICENSE) — free to use, modify, and distribute.
