#!/usr/bin/env bash
set -euo pipefail

export GOROOT="${GOROOT:-$HOME/go}"
export GOPATH="${GOPATH:-$HOME/.local/go}"
export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
OUTDIR="dist"

mkdir -p "$OUTDIR"

echo "Building tidy $VERSION..."

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-linux-amd64" ./cmd/tidy/
echo "  ✓ tidy-linux-amd64 (static)"

GOOS=windows GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy.exe" ./cmd/tidy/
echo "  ✓ tidy.exe"

GOOS=darwin GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-macos-intel" ./cmd/tidy/
echo "  ✓ tidy-macos-intel"

GOOS=darwin GOARCH=arm64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-macos-apple-silicon" ./cmd/tidy/
echo "  ✓ tidy-macos-apple-silicon"

echo ""
echo "All binaries built in $OUTDIR/"
ls -lh "$OUTDIR/"
