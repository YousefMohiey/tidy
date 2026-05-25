#!/usr/bin/env bash
set -euo pipefail

# Usage check
if [[ $# -lt 3 ]]; then
  echo "Usage: $0    [file2.exe ...]" >&2
  exit 1
fi

CERT="$1"
KEY="$2"
shift 2

# Validate cert and key exist
if [[ ! -f "$CERT" ]]; then
  echo "Error: Certificate file not found: $CERT" >&2
  exit 1
fi

if [[ ! -f "$KEY" ]]; then
  echo "Error: Key file not found: $KEY" >&2
  exit 1
fi

# Check for osslsigncode
if ! command -v osslsigncode &>/dev/null; then
  echo "Error: osslsigncode not found." >&2
  echo "" >&2
  echo "Install instructions:" >&2
  echo "  macOS:  brew install osslsigncode" >&2
  echo "  Fedora: sudo dnf install osslsigncode" >&2
  echo "  Ubuntu: sudo apt install osslsigncode" >&2
  exit 1
fi

SIGNED=0
FAILED=0

for INPUT in "$@"; do
  if [[ ! -f "$INPUT" ]]; then
    echo "✗ File not found: $INPUT" >&2
    FAILED=$((FAILED + 1))
    continue
  fi

  BASENAME="${INPUT%.exe}"
  OUTPUT="${BASENAME}-signed.exe"

  echo "Signing: $INPUT → $OUTPUT"

  if osslsigncode sign \
    -certs "$CERT" \
    -key "$KEY" \
    -h sha256 \
    -n "tidy" \
    -i "https://github.com/YousefMohiey/tidy" \
    -ts "http://timestamp.sectigo.com" \
    -in "$INPUT" \
    -out "$OUTPUT" 2>&1; then
    echo "✓ Signed: $OUTPUT"
    SIGNED=$((SIGNED + 1))
  else
    echo "✗ Failed to sign: $INPUT" >&2
    FAILED=$((FAILED + 1))
  fi
done

echo ""
echo "Summary: $SIGNED signed, $FAILED failed"

if [[ $FAILED -gt 0 ]]; then
  exit 1
fi

exit 0
