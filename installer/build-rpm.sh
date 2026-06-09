#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
VERSION="${1:-1.2.0}"

rm -rf /tmp/tidy-rpm
mkdir -p /tmp/tidy-rpm/{SOURCES,SPECS}
cp "$PROJECT_ROOT/dist/tidy-linux-amd64" /tmp/tidy-rpm/SOURCES/tidy-Linux-amd64
cp "$SCRIPT_DIR/tidy.png" /tmp/tidy-rpm/SOURCES/tidy.png
cp "$SCRIPT_DIR/tidy.spec" /tmp/tidy-rpm/SPECS/tidy.spec
sed -i "s/Version:.*/Version:        $VERSION/" /tmp/tidy-rpm/SPECS/tidy.spec
sed -i "s|Source0:.*|Source0:        tidy-Linux-amd64|" /tmp/tidy-rpm/SPECS/tidy.spec

podman run --rm \
    -v /tmp/tidy-rpm:/root/rpmbuild:z \
    -v "$PROJECT_ROOT/dist:/out:z" \
    docker.io/library/fedora:latest \
    bash -c 'dnf install -y rpm-build > /dev/null 2>&1 && rpmbuild -bb --define "_topdir /root/rpmbuild" /root/rpmbuild/SPECS/tidy.spec && cp /root/rpmbuild/RPMS/x86_64/tidy-*.rpm /out/tidy-'"$VERSION"'-1.x86_64.rpm' 2>/dev/null

RPM="$PROJECT_ROOT/dist/tidy-${VERSION}-1.x86_64.rpm"
[ -f "$RPM" ] && echo "  ✓ tidy-${VERSION}-1.x86_64.rpm" || echo "  ✗ RPM build failed"
rm -rf /tmp/tidy-rpm
