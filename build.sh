#!/usr/bin/env bash
set -euo pipefail

export GOROOT="${GOROOT:-$HOME/go}"
export GOPATH="${GOPATH:-$HOME/.local/go}"
export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
CLEAN_VER="${VERSION#v}"
CLEAN_VER="${CLEAN_VER%%-*}"
[ -z "$CLEAN_VER" ] && CLEAN_VER="0.0.0"
OUTDIR="dist"

mkdir -p "$OUTDIR"

echo "Building tidy $VERSION (clean: $CLEAN_VER)..."

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-linux-amd64" ./cmd/tidy/
echo "  ✓ tidy-linux-amd64 (static)"

(
    cd cmd/tidy
    # Prevent stale icon embedding in Windows builds
    rm -f resource.syso
    MAJOR=0; MINOR=0; PATCH=0
    if [[ "$CLEAN_VER" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
        MAJOR="${BASH_REMATCH[1]}"; MINOR="${BASH_REMATCH[2]}"; PATCH="${BASH_REMATCH[3]}"
    fi
    jq --arg maj "$MAJOR" --arg min "$MINOR" --arg pat "$PATCH" --arg ver "$CLEAN_VER" \
        '.FixedFileInfo.FileVersion.Major = ($maj|tonumber) |
         .FixedFileInfo.FileVersion.Minor = ($min|tonumber) |
         .FixedFileInfo.FileVersion.Patch = ($pat|tonumber) |
         .FixedFileInfo.ProductVersion.Major = ($maj|tonumber) |
         .FixedFileInfo.ProductVersion.Minor = ($min|tonumber) |
         .FixedFileInfo.ProductVersion.Patch = ($pat|tonumber) |
         .StringFileInfo.FileVersion = $ver |
         .StringFileInfo.ProductVersion = $ver' \
        versioninfo.json > versioninfo.json.tmp && mv versioninfo.json.tmp versioninfo.json
    goversioninfo -o=resource.syso -64 versioninfo.json
)

GOOS=windows GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-windows-amd64.exe" ./cmd/tidy/
echo "  ✓ tidy-windows-amd64.exe"

./installer/build-installer.sh "$OUTDIR/tidy-windows-amd64.exe" "$OUTDIR/tidy-setup-${CLEAN_VER}.exe" "$CLEAN_VER"

GOOS=darwin GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-macos-intel" ./cmd/tidy/
echo "  ✓ tidy-macos-intel"

GOOS=darwin GOARCH=arm64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-macos-arm64" ./cmd/tidy/
echo "  ✓ tidy-macos-arm64"

./self-sign.sh "$OUTDIR/tidy-windows-amd64.exe" "$OUTDIR/tidy-setup-${CLEAN_VER}.exe"

./installer/build-deb.sh "$CLEAN_VER"
./installer/build-rpm.sh "$CLEAN_VER"

echo ""
echo "All binaries built in $OUTDIR/"
ls -lh "$OUTDIR/"
