#!/bin/bash
# Wrapper to call the python ONNX upscaler with arguments passed from the Go application
# Ensure dependencies are installed in the local .venv

# Get the directory of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$DIR")"
VENV_PYTHON="$PROJECT_ROOT/.venv/bin/python"

# Execute python script
"$VENV_PYTHON" "$PROJECT_ROOT/cmd/python-onnx/main.py" "$@"
