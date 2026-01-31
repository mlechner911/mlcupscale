#!/bin/bash


set -e
# Try to source user profile for PATH (zsh or bash)
if [ -f "$HOME/.zprofile" ]; then
	source "$HOME/.zprofile"
elif [ -f "$HOME/.bash_profile" ]; then
	source "$HOME/.bash_profile"
fi

echo "PATH at build start: $PATH"
which go || true
go version || true

# Variables
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
APP_NAME="MLCupscale"
BUNDLE_ID="com.mlcupscale.app"
VERSION=$(cat ../../VERSION)
BUILD_DIR="$HOME/mlcupscale_build"
APP_DIR="$BUILD_DIR/$APP_NAME.app"
CONTENTS_DIR="$APP_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"

# Clean build dir
rm -rf "$BUILD_DIR"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"

# Download Mac real-esrgan binary and models
echo "Downloading Real-ESRGAN for macOS..."
curl -L -o "$BUILD_DIR/realesrgan.zip" "https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesrgan-ncnn-vulkan-20220424-macos.zip"
unzip -q "$BUILD_DIR/realesrgan.zip" -d "$BUILD_DIR/realesrgan"

# Setup bin and models in Bundle
mkdir -p "$MACOS_DIR/bin" "$MACOS_DIR/models"
cp "$BUILD_DIR/realesrgan/realesrgan-ncnn-vulkan" "$MACOS_DIR/bin/"
chmod +x "$MACOS_DIR/bin/realesrgan-ncnn-vulkan"

# Copy models (from downloaded zip or project - the zip includes them)
cp "$BUILD_DIR/realesrgan/models/"* "$MACOS_DIR/models/"

# Build Go binary (assumes Go is installed)
cd "$SCRIPT_DIR/../.."
# We build as 'mlcupscale-bin' and use a wrapper script as the main executable
go build -ldflags="-s -w -X upscale-service/internal/version.Version=$VERSION" -o "$MACOS_DIR/mlcupscale-bin" ./cmd/server/main.go

# Copy Config
cp config/config.yaml "$MACOS_DIR/config.yaml"

# Create Wrapper Script to set CWD
cat <<EOF > "$MACOS_DIR/mlcupscale"
#!/bin/bash
cd "\$(dirname "\$0")"

# Set storage paths to user home to avoid writing into App Bundle
export UPSCALE_STORAGE_UPLOAD_DIR="\$HOME/.mlcupscale/data/uploads"
export UPSCALE_STORAGE_OUTPUT_DIR="\$HOME/.mlcupscale/data/outputs"

# Create directories
mkdir -p "\$UPSCALE_STORAGE_UPLOAD_DIR"
mkdir -p "\$UPSCALE_STORAGE_OUTPUT_DIR"

# Log file
LOG_FILE="\$HOME/Library/Logs/MLCupscale.log"
echo "Starting MLCupscale at \$(date)" >> "\$LOG_FILE"

./mlcupscale-bin -config config.yaml >> "\$LOG_FILE" 2>&1
EOF
chmod +x "$MACOS_DIR/mlcupscale"

# Copy Info.plist
cp "$SCRIPT_DIR/Info.plist" "$CONTENTS_DIR/Info.plist"

# Optionally copy icons/resources here

# Create DMG
cd "$BUILD_DIR"
hdiutil create -volname "$APP_NAME" -srcfolder "$APP_NAME.app" -ov -format UDZO "$APP_NAME.dmg"

echo "DMG created at $BUILD_DIR/$APP_NAME.dmg"
