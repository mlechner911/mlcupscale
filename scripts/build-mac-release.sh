#!/bin/bash
set -e

# Configuration
APP_NAME="upscale-service"
DIST_DIR="dist/macos"
BIN_NAME="upscale-server"

echo "Building MacOS Release..."

# 1. Clean and Create Dist Directory
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR/bin"
mkdir -p "$DIST_DIR/config"
mkdir -p "$DIST_DIR/cmd/python-onnx"
mkdir -p "$DIST_DIR/models"
mkdir -p "$DIST_DIR/scripts"

# 2. Cross-Compile Go Server for MacOS (Apple Silicon & Intel)
# Using CGO_ENABLED=0 since dependencies look pure Go or we don't need CGO for this simple service
echo "Compiling Go binary for macOS (arm64)..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o "$DIST_DIR/$BIN_NAME" ./cmd/server/main.go

# 3. Copy Python Source & Requirements
echo "Copying Python resources..."
cp cmd/python-onnx/main.py "$DIST_DIR/cmd/python-onnx/"
cp cmd/python-onnx/convert.py "$DIST_DIR/cmd/python-onnx/"
cp cmd/python-onnx/requirements.txt "$DIST_DIR/cmd/python-onnx/"

# 4. Copy Wrapper Script
echo "Copying Helper scripts..."
cp scripts/upscale-onnx.sh "$DIST_DIR/scripts/"
chmod +x "$DIST_DIR/scripts/upscale-onnx.sh"

# 5. Copy Config (Template)
echo "Copying Configuration..."
cp config/config.yaml "$DIST_DIR/config/config.example.yaml"

# 6. Copy ONNX Models (if they exist)
# We only assume the user wants the converted ones
echo "Copying ONNX models..."
if ls models/*.onnx 1> /dev/null 2>&1; then
    cp models/*.onnx "$DIST_DIR/models/"
fi

# 7. Create Setup Script for the destination Mac
cat <<EOF > "$DIST_DIR/setup_mac.sh"
#!/bin/bash
set -e

echo "=== ML Upscale Service Setup for macOS ==="

# Check for Python 3
if ! command -v python3 &> /dev/null; then
    echo "Error: python3 could not be found. Please install it (e.g., install python from swift.org or use brew)."
    exit 1
fi

echo "Creating Python virtual environment..."
python3 -m venv .venv

echo "Activating venv and installing dependencies..."
source .venv/bin/activate

# Install dependencies
# Note: modern pip/torch usually handles M1/M2 chips automatically
pip install -r cmd/python-onnx/requirements.txt

echo "Setup complete!"
echo "To run the server:"
echo "1. Rename config: cp config/config.example.yaml config/config.yaml"
echo "2. Edit config.yaml to set 'binary_path' to './scripts/upscale-onnx.sh'"
echo "3. Run: ./$BIN_NAME"
EOF
chmod +x "$DIST_DIR/setup_mac.sh"

# 8. Archive
echo "Creating Zip archive..."
cd dist
zip -r "upscale-service-macos.zip" macos
cd ..

echo "Done! Release package is at dist/upscale-service-macos.zip"
