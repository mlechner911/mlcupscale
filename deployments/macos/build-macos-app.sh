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

# ---------------------------------------------------------
# PYTHON / ONNX SETUP (PyInstaller)
# ---------------------------------------------------------

echo "[1/6] Setting up Build Environment & Freezing Python..."

# 1. Create a temporary build venv on the Build Mac
# We need this to have a clean environment for PyInstaller to harvest packages from
echo "      - Creating temporary venv in $BUILD_DIR/venv..."
python3 -m venv "$BUILD_DIR/venv"
source "$BUILD_DIR/venv/bin/activate"

# 2. Install Dependencies + PyInstaller
# Ensure we have wheel to avoid build issues
echo "      - Upgrading pip & wheel..."
pip install --upgrade pip wheel > /dev/null

echo "      - Installing Dependencies (this may take a minute)..."
pip install -r ../../cmd/python-onnx/requirements.txt

echo "      - Installing PyInstaller..."
pip install pyinstaller > /dev/null

# 3. Build Standalone Executable
# --onedir: Bundles everything into a directory (faster startup than --onefile)
# --name: output name
# --windowed: (Optional) Suppresses console window if we were a GUI app
echo "[2/6] Running PyInstaller (Freezing Python)..."
echo "      - This step can take 2-5 minutes. Please wait..."
pyinstaller --clean --onedir \
    --name upscale-onnx \
    --distpath "$MACOS_DIR/bin" \
    --workpath "$BUILD_DIR/pyinstaller_work" \
    --specpath "$BUILD_DIR" \
    --log-level INFO \
    ../../cmd/python-onnx/main.py

# Clean up venv
deactivate

# 4. Copy Models (ONNX)
echo "[3/6] Copying Models..."
mkdir -p "$MACOS_DIR/models"
cp ../../models/*.onnx "$MACOS_DIR/models/" || echo "      ! Warning: No .onnx models found to copy!"
# Also copy external data files if they exist
cp ../../models/*.onnx.data "$MACOS_DIR/models/" 2>/dev/null || true

# ---------------------------------------------------------
# GO BUILD
# ---------------------------------------------------------

# Build Go binary (assumes Go is installed)
echo "[4/6] Compiling Go Server..."
cd "$SCRIPT_DIR/../.."
# We build as 'mlcupscale-bin' and use a wrapper script as the main executable
CGO_ENABLED=0 go build -ldflags="-s -w -X upscale-service/internal/version.Version=$VERSION" -o "$MACOS_DIR/mlcupscale-bin" ./cmd/server/main.go

# ---------------------------------------------------------
# CONFIGURATION
# ---------------------------------------------------------

# Copy Config and modify it for the App Bundle paths
echo "[5/6] Configuration & Finalizing Bundle..."
cp config/config.yaml "$MACOS_DIR/config.yaml"

# Update binary_path to point to our new compiled binary
# Note: PyInstaller with --onedir creates the binary at ./bin/upscale-onnx/upscale-onnx
# Use temp file instead of sed -i to avoid BSD/GNU sed differences on MacOS (user might have gnu-sed from brew)
sed 's|binary_path:.*|binary_path: "./bin/upscale-onnx/upscale-onnx"|' "$MACOS_DIR/config.yaml" > "$MACOS_DIR/config.yaml.tmp" && mv "$MACOS_DIR/config.yaml.tmp" "$MACOS_DIR/config.yaml"
sed 's|models_path:.*|models_path: "./models"|' "$MACOS_DIR/config.yaml" > "$MACOS_DIR/config.yaml.tmp" && mv "$MACOS_DIR/config.yaml.tmp" "$MACOS_DIR/config.yaml"

# ---------------------------------------------------------
# APP LAUNCHER WRAPPER
# ---------------------------------------------------------

# Create Wrapper Script to set CWD and Launch
cat <<EOF > "$MACOS_DIR/mlcupscale"
#!/bin/bash
cd "\$(dirname "\$0")"

# --- 1. Environment Setup ---
export UPSCALE_STORAGE_UPLOAD_DIR="\$HOME/.mlcupscale/data/uploads"
export UPSCALE_STORAGE_OUTPUT_DIR="\$HOME/.mlcupscale/data/outputs"

mkdir -p "\$UPSCALE_STORAGE_UPLOAD_DIR"
mkdir -p "\$UPSCALE_STORAGE_OUTPUT_DIR"

# Log file
LOG_FILE="\$HOME/Library/Logs/MLCupscale.log"
echo "Starting MLCupscale at \$(date)" >> "\$LOG_FILE"

# --- 2. Run Server ---
# The server will call ./bin/upscale-onnx which is now a self-contained binary
./mlcupscale-bin -config config.yaml >> "\$LOG_FILE" 2>&1
EOF
chmod +x "$MACOS_DIR/mlcupscale"

# Copy Info.plist
cp "$SCRIPT_DIR/Info.plist" "$CONTENTS_DIR/Info.plist"

# Optionally copy icons/resources here

# Create DMG
echo "[6/6] Creating DMG Image..."
cd "$BUILD_DIR"
hdiutil create -volname "$APP_NAME" -srcfolder "$APP_NAME.app" -ov -format UDZO "$APP_NAME.dmg"

echo "✅ Success! DMG created at $BUILD_DIR/$APP_NAME.dmg"
