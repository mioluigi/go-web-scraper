#!/usr/bin/env bash
# build.sh — executables for macOS (Apple Silicon and Intel) and Windows, in dist/.
# Pure Go (no C), so this cross-compiles from any machine. The .app is built by
# scripts/build-macos-app.sh.
set -euo pipefail
cd "$(dirname "$0")/.."
mkdir -p dist
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -o dist/go-web-scraper-macos-arm64 .
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -trimpath -o dist/go-web-scraper-macos-amd64 .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -o dist/go-web-scraper.exe .
ls -l dist
