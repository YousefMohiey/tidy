#!/usr/bin/env bash
set -euo pipefail

export GOROOT="${GOROOT:-$HOME/go}"
export GOPATH="${GOPATH:-$HOME/.local/go}"
export PATH="$GOROOT/bin:$GOPATH/bin:$PATH"

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
OUTDIR="dist"

mkdir -p "$OUTDIR"

echo "Building tidy $VERSION..."

GOOS=linux GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-linux-amd64" ./cmd/tidy/
echo "  ✓ tidy-linux-amd64"

GOOS=windows GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-windows-amd64.exe" ./cmd/tidy/
echo "  ✓ tidy-windows-amd64.exe"

GOOS=darwin GOARCH=amd64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-darwin-amd64" ./cmd/tidy/
echo "  ✓ tidy-darwin-amd64"

GOOS=darwin GOARCH=arm64 go build -buildvcs=false -ldflags="-s -w" -o "$OUTDIR/tidy-darwin-arm64" ./cmd/tidy/
echo "  ✓ tidy-darwin-arm64"

echo ""
echo "All binaries built in $OUTDIR/"
ls -lh "$OUTDIR/"
