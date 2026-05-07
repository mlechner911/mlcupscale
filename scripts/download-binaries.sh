#!/bin/bash
set -e

# Target directory
BIN_DIR="bin"
mkdir -p "$BIN_DIR"

# Download NCNN binaries for different platforms
# v0.2.5.0 is a stable version for these binaries

echo "=== Downloading Upscaler Binaries ==="

# Linux
if [ ! -f "$BIN_DIR/realesrgan-ncnn-vulkan" ]; then
    echo "Downloading Linux binary..."
    TEMP_DIR=$(mktemp -d)
    wget -qO "$TEMP_DIR/linux.zip" "https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesrgan-ncnn-vulkan-20220424-ubuntu.zip"
    unzip -qo "$TEMP_DIR/linux.zip" -d "$TEMP_DIR"
    cp "$TEMP_DIR/realesrgan-ncnn-vulkan" "$BIN_DIR/realesrgan-ncnn-vulkan"
    chmod +x "$BIN_DIR/realesrgan-ncnn-vulkan"
    rm -rf "$TEMP_DIR"
fi

# Windows
if [ ! -f "$BIN_DIR/realesrgan-ncnn-vulkan.exe" ]; then
    echo "Downloading Windows binary..."
    TEMP_DIR=$(mktemp -d)
    wget -qO "$TEMP_DIR/windows.zip" "https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesrgan-ncnn-vulkan-20220424-windows.zip"
    unzip -qo "$TEMP_DIR/windows.zip" -d "$TEMP_DIR"
    cp "$TEMP_DIR/realesrgan-ncnn-vulkan.exe" "$BIN_DIR/realesrgan-ncnn-vulkan.exe"
    rm -rf "$TEMP_DIR"
fi

# macOS
if [ ! -f "$BIN_DIR/realesrgan-ncnn-vulkan-macos" ]; then
    echo "Downloading macOS binary..."
    TEMP_DIR=$(mktemp -d)
    wget -qO "$TEMP_DIR/macos.zip" "https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesrgan-ncnn-vulkan-20220424-macos.zip"
    unzip -qo "$TEMP_DIR/macos.zip" -d "$TEMP_DIR"
    cp "$TEMP_DIR/realesrgan-ncnn-vulkan" "$BIN_DIR/realesrgan-ncnn-vulkan-macos"
    chmod +x "$BIN_DIR/realesrgan-ncnn-vulkan-macos"
    rm -rf "$TEMP_DIR"
fi

echo "=== Binaries downloaded ==="
ls -lh "$BIN_DIR"
