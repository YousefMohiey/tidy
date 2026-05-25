#!/usr/bin/env bash
set -euo pipefail

export GOROOT="${GOROOT:-$HOME/go}"
export GOPATH="${GOPATH:-$HOME/.local/go}"
export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
OUTDIR="dist"

mkdir -p "$OUTDIR"

echo "Building tidy $VERSION..."

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-Linux-amd64" ./cmd/tidy/
echo "  ✓ tidy-Linux-amd64 (static)"

(
    cd cmd/tidy
    CLEAN_VERSION="${VERSION#v}"
    CLEAN_VERSION="${CLEAN_VERSION%-dirty}"
    
    MAJOR=0
    MINOR=0
    PATCH=0
    if [[ "$CLEAN_VERSION" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
        MAJOR="${BASH_REMATCH[1]}"
        MINOR="${BASH_REMATCH[2]}"
        PATCH="${BASH_REMATCH[3]}"
    fi
    
    jq --arg maj "$MAJOR" --arg min "$MINOR" --arg pat "$PATCH" --arg ver "$VERSION" \
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

GOOS=windows GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-Windows-amd64.exe" ./cmd/tidy/
echo "  ✓ tidy-Windows-amd64.exe (with version info)"

./installer/build-installer.sh "$OUTDIR/tidy-Windows-amd64.exe" "$OUTDIR/tidy-Setup-$VERSION.exe" "$VERSION"

GOOS=darwin GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-macOS-Intel" ./cmd/tidy/
echo "  ✓ tidy-macOS-Intel"

GOOS=darwin GOARCH=arm64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-macOS-Apple-Silicon" ./cmd/tidy/
echo "  ✓ tidy-macOS-Apple-Silicon"

./self-sign.sh "$OUTDIR/tidy-Windows-amd64.exe" "$OUTDIR/tidy-Setup-$VERSION.exe"

echo ""
echo "All binaries built in $OUTDIR/"
ls -lh "$OUTDIR/"
