#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
VERSION="${1:-1.2.0}"

podman run --rm \
    -v "$PROJECT_ROOT:/work:z" -w /work \
    docker.io/library/ubuntu:latest \
    bash -c '
        set -e
        apt-get update -qq && apt-get install -y -qq dpkg-dev > /dev/null 2>&1
        mkdir -p pkg/usr/bin pkg/DEBIAN pkg/usr/share/icons/hicolor/256x256/apps pkg/usr/share/applications
        cp dist/tidy-linux-amd64 pkg/usr/bin/tidy && chmod 755 pkg/usr/bin/tidy
        cp installer/tidy.png pkg/usr/share/icons/hicolor/256x256/apps/tidy.png
        cat > pkg/DEBIAN/control << CTRL
Package: tidy
Version: '"$VERSION"'
Section: utils
Priority: optional
Architecture: amd64
Maintainer: YousefMohiey <yousefmohiey@gmail.com>
Homepage: https://github.com/YousefMohiey/tidy
Description: Smart file organizer for your terminal
 tidy organizes files into categorized directories by content type.
 Features interactive TUI, duplicate detection, watch mode, and full undo.
CTRL
        cat > pkg/usr/share/applications/tidy.desktop << DSK
[Desktop Entry]
Name=Tidy
Comment=Smart file organizer
Exec=tidy
Icon=tidy
Terminal=true
Type=Application
Categories=Utility;FileManager;
DSK
        dpkg-deb --build pkg dist/tidy-'"$VERSION"'-amd64.deb
    ' 2>/dev/null

DEB="$PROJECT_ROOT/dist/tidy-${VERSION}-amd64.deb"
[ -f "$DEB" ] && echo "  ✓ tidy-${VERSION}-amd64.deb" || echo "  ✗ DEB build failed"
