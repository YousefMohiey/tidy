#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="$SCRIPT_DIR/tidy"
INSTALL_DIR="${HOME}/.local/bin"
DESKTOP_DIR="${HOME}/.local/share/applications"
ICON_DIR="${HOME}/.local/share/icons/hicolor/256x256/apps"

echo "Installing tidy..."

mkdir -p "$INSTALL_DIR"
cp "$BINARY" "$INSTALL_DIR/tidy"
chmod +x "$INSTALL_DIR/tidy"
echo "  Binary installed to $INSTALL_DIR/tidy"

mkdir -p "$ICON_DIR"
if [ -f "$SCRIPT_DIR/installer/tidy.png" ]; then
    cp "$SCRIPT_DIR/installer/tidy.png" "$ICON_DIR/tidy.png"
    echo "  Icon installed to $ICON_DIR/tidy.png"
fi

mkdir -p "$DESKTOP_DIR"
cat > "$DESKTOP_DIR/tidy.desktop" << 'EOF'
[Desktop Entry]
Name=Tidy
Comment=Smart file organizer
Exec=tidy
Icon=tidy
Terminal=true
Type=Application
Categories=Utility;FileManager;
Keywords=files;organize;sort;cleanup;
EOF

if [ -x "$(command -v gtk-update-icon-cache)" ]; then
    gtk-update-icon-cache -f "$HOME/.local/share/icons/hicolor" 2>/dev/null || true
fi

if [ -x "$(command -v update-desktop-database)" ]; then
    update-desktop-database "$HOME/.local/share/applications" 2>/dev/null || true
fi

echo "  Desktop entry installed to $DESKTOP_DIR/tidy.desktop"
echo ""
echo "Done! Run 'tidy' from terminal or find it in your application menu."
