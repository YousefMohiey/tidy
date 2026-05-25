#!/usr/bin/env bash
set -euo pipefail

# self-sign.sh — Generate a self-signed code signing certificate and sign Windows binaries.
# Uses openssl for cert generation and osslsigncode (via podman) for signing.
#
# Usage:
#   ./self-sign.sh                          # Sign all .exe in dist/
#   ./self-sign.sh dist/tidy-Windows-amd64.exe dist/tidy-Setup-*.exe
#
# The certificate is stored in .certs/ (gitignored). Re-running reuses the existing cert.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERT_DIR="$SCRIPT_DIR/.certs"
CERT_FILE="$CERT_DIR/tidy-selfsigned.pem"
KEY_FILE="$CERT_DIR/tidy-selfsigned.key"
CERT_SUBJECT="/CN=YousefMohiey/O=YousefMohiey/C=US"
CERT_DAYS=1825  # 5 years

mkdir -p "$CERT_DIR"

# Generate cert if it doesn't exist
if [[ ! -f "$CERT_FILE" || ! -f "$KEY_FILE" ]]; then
    echo "Generating self-signed code signing certificate..."
    openssl req -x509 -newkey rsa:2048 -nodes \
        -keyout "$KEY_FILE" \
        -out "$CERT_FILE" \
        -days "$CERT_DAYS" \
        -subj "$CERT_SUBJECT" \
        -addext "extendedKeyUsage=codeSigning" \
        2>/dev/null
    echo "  ✓ Certificate generated in .certs/"
else
    echo "  ✓ Using existing certificate from .certs/"
fi

# Collect files to sign
FILES=()
if [[ $# -gt 0 ]]; then
    FILES=("$@")
else
    for f in "$SCRIPT_DIR"/dist/*.exe; do
        [[ -f "$f" ]] && FILES+=("$f")
    done
fi

if [[ ${#FILES[@]} -eq 0 ]]; then
    echo "No .exe files found to sign."
    exit 0
fi

echo "Signing ${#FILES[@]} file(s)..."

SIGNED=0
FAILED=0

for INPUT in "${FILES[@]}"; do
    if [[ ! -f "$INPUT" ]]; then
        echo "  ✗ Not found: $INPUT"
        FAILED=$((FAILED + 1))
        continue
    fi

    BASENAME="$(basename "$INPUT")"
    REL_INPUT="${INPUT#$SCRIPT_DIR/}"
    REL_SIGNED="${REL_INPUT}.signed"

    # Sign using osslsigncode via podman
    if podman run --rm \
        -v "$SCRIPT_DIR:/work:z" \
        -w /work \
        docker.io/library/fedora:latest \
        bash -c "dnf install -y osslsigncode > /dev/null 2>&1 && \
            osslsigncode sign \
                -certs .certs/tidy-selfsigned.pem \
                -key .certs/tidy-selfsigned.key \
                -h sha256 \
                -n tidy \
                -i https://github.com/YousefMohiey/tidy \
                -in $REL_INPUT \
                -out $REL_SIGNED" 2>/dev/null; then
        mv "$SCRIPT_DIR/$REL_SIGNED" "$INPUT"
        echo "  ✓ Signed: $BASENAME"
        SIGNED=$((SIGNED + 1))
    else
        echo "  ✗ Failed: $BASENAME"
        rm -f "$SIGNED_FILE"
        FAILED=$((FAILED + 1))
    fi
done

echo ""
echo "Self-signing complete: $SIGNED signed, $FAILED failed"
if [[ $FAILED -gt 0 ]]; then
    exit 1
fi
