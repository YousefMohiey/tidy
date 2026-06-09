#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BINARY="${1:-$PROJECT_ROOT/dist/tidy-Linux-amd64}"
VERSION="${2:-1.1.0}"
OUTDIR="$PROJECT_ROOT/dist"

mkdir -p "$OUTDIR"

cp "$BINARY" /tmp/tidy-deb-binary
cp "$SCRIPT_DIR/tidy.png" /tmp/tidy-deb-icon.png 2>/dev/null || touch /tmp/tidy-deb-icon.png

podman run --rm \
    -v /tmp/tidy-deb-binary:/work/tidy-linux:z \
    -v /tmp/tidy-deb-icon.png:/work/tidy.png:z \
    -v "$OUTDIR:/out:z" \
    docker.io/library/ubuntu:latest \
    bash -c "
        apt-get update -qq && apt-get install -y -qq dpkg-dev > /dev/null 2>&1 &&
        mkdir -p pkg/usr/bin pkg/DEBIAN pkg/usr/share/icons/hicolor/256x256/apps pkg/usr/share/applications &&
        cp tidy-linux pkg/usr/bin/tidy && chmod 755 pkg/usr/bin/tidy &&
        cp tidy.png pkg/usr/share/icons/hicolor/256x256/apps/tidy.png 2>/dev/null || true &&
        cat > pkg/DEBIAN/control << 'CTRL'
Package: tidy
Version: $VERSION
Section: utils
Priority: optional
Architecture: amd64
Maintainer: YousefMohiey <yousefmohiey@gmail.com>
Homepage: https://github.com/YousefMohiey/tidy
Description: Smart file organizer for your terminal
 tidy organizes files into categorized directories by content type.
 Features interactive TUI, duplicate detection, watch mode, and full undo.
CTRL
        cat > pkg/usr/share/applications/tidy.desktop << 'DSK'
[Desktop Entry]
Name=Tidy
Comment=Smart file organizer
Exec=tidy
Icon=tidy
Terminal=true
Type=Application
Categories=Utility;FileManager;
DSK
        dpkg-deb --build pkg /out/tidy_${VERSION}_amd64.deb
    " 2>/dev/null

rm -f /tmp/tidy-deb-binary /tmp/tidy-deb-icon.png

if [ -f "$OUTDIR/tidy_${VERSION}_amd64.deb" ]; then
    echo "  ✓ tidy_${VERSION}_amd64.deb"
else
    echo "  ✗ DEB build failed (requires podman)"
fi
