#!/usr/bin/env bash
set -euo pipefail

BINARY="tidy"
INSTALL_DIR="${HOME}/.local/bin"
DESKTOP_DIR="${HOME}/.local/share/applications"
ICON_DIR="${HOME}/.local/share/icons/hicolor/256x256/apps"

echo "Installing tidy..."

mkdir -p "$INSTALL_DIR"
cp "$BINARY" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"
echo "  Binary installed to $INSTALL_DIR/$BINARY"

mkdir -p "$DESKTOP_DIR"
cat > "$DESKTOP_DIR/tidy.desktop" << 'EOF'
[Desktop Entry]
Name=Tidy
Comment=Smart file organizer
Exec=bash -c 'tidy' 
Icon=utilities-terminal
Terminal=true
Type=Application
Categories=Utility;FileManager;
Keywords=files;organize;sort;cleanup;
EOF
echo "  Desktop entry installed to $DESKTOP_DIR/tidy.desktop"

echo ""
echo "Done! Run 'tidy' from terminal or find it in your application menu."
