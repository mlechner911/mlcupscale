#!/bin/bash
set -e

echo "=== Building Upscale Service ==="

VERSION=${1:-"1.0.0"}
BUILD_DIR="build"

# Clean
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

echo "Building for multiple platforms..."

# Linux
echo "Building for Linux..."
GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o "$BUILD_DIR/upscale-server-linux-amd64" \
    ./cmd/server

# macOS (Intel)
echo "Building for macOS (Intel)..."
GOOS=darwin GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o "$BUILD_DIR/upscale-server-darwin-amd64" \
    ./cmd/server

# macOS (Apple Silicon)
echo "Building for macOS (ARM)..."
GOOS=darwin GOARCH=arm64 go build \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o "$BUILD_DIR/upscale-server-darwin-arm64" \
    ./cmd/server

# Windows
echo "Building for Windows..."
GOOS=windows GOARCH=amd64 go build \
    -ldflags="-s -w -X main.version=$VERSION" \
    -o "$BUILD_DIR/upscale-server-windows-amd64.exe" \
    ./cmd/server

echo "=== Build complete ==="
ls -lh "$BUILD_DIR"
