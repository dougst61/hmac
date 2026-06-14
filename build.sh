#!/usr/bin/env bash
set -euo pipefail

VERSION=$(cat VERSION)
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILT_BY=$(whoami)

APP_NAME="featuretoken"
BUILD_DIR="build"
LDFLAGS="-X main.Version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE} -X main.builtBy=${BUILT_BY}"

echo "Building ${APP_NAME} v${VERSION} (${COMMIT})"
echo "---"

rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

echo "Building linux/amd64..."
GOOS=linux GOARCH=amd64 GOAMD64=v3 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-linux-amd64" .

echo "Building windows/amd64..."
GOOS=windows GOARCH=amd64 GOAMD64=v3 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-windows-amd64.exe" .

echo "Building darwin/arm64..."
GOOS=darwin GOARCH=arm64 GOARM64=v8.4,lse,crypto go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-darwin-arm64" .

echo "Building darwin/amd64..."
GOOS=darwin GOARCH=amd64 GOAMD64=v3 go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${APP_NAME}-darwin-amd64" .

echo "---"
echo "Build complete. Binaries in ${BUILD_DIR}/:"
ls -lh "${BUILD_DIR}/"
echo "Version: ${VERSION}"
