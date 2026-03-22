#!/usr/bin/env bash
set -euo pipefail

MAJOR=1
MINOR=0
REVISION=0

BUILD=$(date -u +"%y%m%d%H%M")
VERSION="${MAJOR}.${MINOR}.${REVISION}.${BUILD}"

APP_NAME="featuretoken"
BUILD_DIR="build"
LDFLAGS="-X main.Version=${VERSION}"

echo "Building ${APP_NAME} v${VERSION}"
echo "---"

rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Linux amd64
echo "Building linux/amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-linux-amd64" .

# Windows amd64
echo "Building windows/amd64..."
GOOS=windows GOARCH=amd64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-windows-amd64.exe" .

# macOS arm64
echo "Building darwin/arm64..."
GOOS=darwin GOARCH=arm64 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-darwin-arm64" .

echo "---"
echo "Build complete. Binaries in ${BUILD_DIR}/:"
ls -lh "${BUILD_DIR}/"
echo "Version: ${VERSION}"
