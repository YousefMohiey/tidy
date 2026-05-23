#!/usr/bin/env bash
set -euo pipefail

export GOROOT="${GOROOT:-$HOME/go}"
export GOPATH="${GOPATH:-$HOME/.local/go}"
export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
OUTDIR="dist"

mkdir -p "$OUTDIR"

echo "Building tidy $VERSION..."

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/Tidy-Linux-amd64" ./cmd/tidy/
echo "  ✓ Tidy-Linux-amd64 (static)"

GOOS=windows GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/Tidy-Windows-amd64.exe" ./cmd/tidy/
echo "  ✓ Tidy-Windows-amd64.exe"

GOOS=darwin GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/Tidy-macOS-Intel" ./cmd/tidy/
echo "  ✓ Tidy-macOS-Intel"

GOOS=darwin GOARCH=arm64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/Tidy-macOS-Apple-Silicon" ./cmd/tidy/
echo "  ✓ Tidy-macOS-Apple-Silicon"

echo ""
echo "All binaries built in $OUTDIR/"
ls -lh "$OUTDIR/"
