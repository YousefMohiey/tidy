#!/usr/bin/env bash
set -euo pipefail

WINDOWS_BINARY="${1:?Usage: build-installer.sh  }"
INSTALLER_OUTPUT="${2:?Usage: build-installer.sh  }"
VERSION="${3:-0.0.0}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

if [[ ! -f "$WINDOWS_BINARY" ]]; then
    echo "Error: Windows binary not found: $WINDOWS_BINARY"
    exit 1
fi

echo "Building Windows installer..."

INSTALLER_BASENAME="$(basename "$INSTALLER_OUTPUT" .exe)"

CLEAN_VERSION="${VERSION#v}"
CLEAN_VERSION="${CLEAN_VERSION%-dirty}"
NUMERIC_VERSION="0.0.0.0"
if [[ "$CLEAN_VERSION" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
    NUMERIC_VERSION="${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.${BASH_REMATCH[3]}.0"
fi

cp "$WINDOWS_BINARY" "$SCRIPT_DIR/tidy.exe"

cleanup() { rm -f "$SCRIPT_DIR/tidy.exe"; }
trap cleanup EXIT

if command -v makensis &>/dev/null; then
    makensis -V2 \
        -DPRODUCT_VERSION="$VERSION" \
        -DPRODUCT_VERSION_NUMERIC="$NUMERIC_VERSION" \
        -DPRODUCT_INSTALLER_NAME="$INSTALLER_BASENAME" \
        "$SCRIPT_DIR/tidy.nsi"
else
    podman run --rm \
        -v "$PROJECT_ROOT:/work:z" \
        -w /work \
        docker.io/library/fedora:latest \
        bash -c "dnf install -y mingw-nsis-base mingw32-nsis mingw64-nsis > /dev/null 2>&1 && makensis -V2 -DPRODUCT_VERSION='$VERSION' -DPRODUCT_VERSION_NUMERIC='$NUMERIC_VERSION' -DPRODUCT_INSTALLER_NAME='$INSTALLER_BASENAME' installer/tidy.nsi"
fi

if [[ -f "$INSTALLER_OUTPUT" ]]; then
    echo "  ✓ Windows installer: $(basename "$INSTALLER_OUTPUT")"
else
    echo "Error: Installer not created"
    exit 1
fi
