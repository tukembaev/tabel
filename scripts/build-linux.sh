#!/usr/bin/env bash
# Cross-compile production binaries for Linux servers (no Docker required).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${ROOT}/bin"
mkdir -p "$OUT"

GOOS="${GOOS:-linux}"
GOARCH="${GOARCH:-amd64}"
VERSION="${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}"
LDFLAGS="-s -w -X main.version=${VERSION}"

echo "Building tabel-server (${GOOS}/${GOARCH})..."
CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath -ldflags "$LDFLAGS" \
  -o "$OUT/tabel-server" "$ROOT/cmd/server"

echo "Building tabel-worker (${GOOS}/${GOARCH})..."
CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath -ldflags "$LDFLAGS" \
  -o "$OUT/tabel-worker" "$ROOT/cmd/worker"

echo "Done:"
ls -lh "$OUT/tabel-server" "$OUT/tabel-worker"
