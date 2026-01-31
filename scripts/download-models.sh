#!/bin/bash
set -e

MODELS_DIR="models"
TEMP_DIR="/tmp/realesrgan-models"

echo "=== Downloading Real-ESRGAN Models ==="

mkdir -p "$MODELS_DIR"
mkdir -p "$TEMP_DIR"

ROOT_DIR=$(pwd)
ABS_MODELS_DIR="$ROOT_DIR/$MODELS_DIR"

cd "$TEMP_DIR"

# Download model pack
echo "Downloading model pack..."
wget -qO models.zip \
    https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesrgan-ncnn-vulkan-20220424-ubuntu.zip

# Extract
echo "Extracting..."
unzip -qo models.zip

# Copy models
echo "Copying models to $ABS_MODELS_DIR..."
# The zip structure has a 'models' directory at the root
cp -r models/* "$ABS_MODELS_DIR/"

# Copy binary (for local execution)
echo "Copying binary..."
mkdir -p "$ROOT_DIR/bin"
cp realesrgan-ncnn-vulkan "$ROOT_DIR/bin/"
chmod +x "$ROOT_DIR/bin/realesrgan-ncnn-vulkan"

# Cleanup
cd "$ROOT_DIR"
rm -rf "$TEMP_DIR"

echo "=== Models installed ==="
ls -lh "$MODELS_DIR"