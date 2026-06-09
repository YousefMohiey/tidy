#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BINARY="${1:-$PROJECT_ROOT/dist/tidy-Linux-amd64}"
VERSION="${2:-1.1.0}"
SPEC="$SCRIPT_DIR/tidy.spec"

if ! command -v rpmbuild &>/dev/null; then
    echo "rpmbuild not found. Use: sudo dnf install rpm-build"
    exit 1
fi

RPM_TOPDIR="$(mktemp -d)"
mkdir -p "$RPM_TOPDIR"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
cp "$BINARY" "$RPM_TOPDIR/SOURCES/tidy-Linux-amd64"
cp "$SCRIPT_DIR/tidy.png" "$RPM_TOPDIR/SOURCES/tidy.png"
sed "s/Version:.*/Version:        $VERSION/" "$SPEC" > "$RPM_TOPDIR/SPECS/tidy.spec"

rpmbuild -bb --define "_topdir $RPM_TOPDIR" "$RPM_TOPDIR/SPECS/tidy.spec"
RPM_FILE=$(find "$RPM_TOPDIR/RPMS" -name "tidy-*.rpm" | head -1)
if [ -f "$RPM_FILE" ]; then
    cp "$RPM_FILE" "$PROJECT_ROOT/dist/"
    echo "  ✓ $(basename "$RPM_FILE")"
fi
rm -rf "$RPM_TOPDIR"
