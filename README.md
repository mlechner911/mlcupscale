# Image Upscale Service

A high-performance, containerized REST API for AI-based image upscaling using Real-ESRGAN (ncnn-vulkan).

![Go Version](https://img.shields.io/badge/Go-1.24-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![Status](https://img.shields.io/badge/Status-Beta-orange)

## Features

*   **AI Upscaling**: High-quality 2x, 3x, and 4x image upscaling.
*   **Models**: Includes `realesrgan-x4plus`, `realesrgan-x4plus-anime`, and `realesr-animevideov3`.
*   **Performance**: Optimized for GPU (Vulkan) with CPU fallback.
*   **API**: Modern, asynchronous REST API with job tracking and swagger documentation.
*   **Production Ready**: Docker support, health checks, metrics, and rate limiting.

## Prerequisites

*   **Linux/macOS** (Windows requires WSL2)
*   **Go 1.24+** (for building locally)
*   **Task** (recommended) or **Make**
*   **Docker & Docker Compose** (optional, for containerized deployment)
*   **Vulkan Driver** (optional, for GPU acceleration)

## Getting Started

### 1. Clone the Repository

```bash
git clone <repository-url>
cd mlcupscale
```

### 2. Download Models & Binaries

This step is required for both local and Docker usage. It downloads the Real-ESRGAN models and the `realesrgan-ncnn-vulkan` binary from the [upstream repository](https://github.com/xinntao/Real-ESRGAN/releases).

**Note on the Binary:**
The default setup uses pre-compiled binaries for convenience. If you need to build `realesrgan-ncnn-vulkan` from source (e.g., for optimized Vulkan shaders, specific hardware support, or non-standard architectures), please follow the [Real-ESRGAN Build Instructions](https://github.com/xinntao/Real-ESRGAN#dependencies). After building, place your custom executable in the `bin/` directory (or `bin/` inside the app bundle for macOS).

Using Task:
```bash
task download-models
```

Using Make:
```bash
make models
```

### 3. Run Locally

Build and start the server:

Using Task:
```bash
task run
```

Using Make:
```bash
make run
```

The server will start at `http://localhost:8089`.

### 4. Run with Docker

Build and run using Docker Compose:

```bash
make docker-run
```

To stop the service:
```bash
make docker-stop
```

### 5. Build for macOS (Remote)

If you have SSH access to a macOS machine (e.g., Apple Silicon), you can build a native `.app` and `.dmg` remotely.

Prerequisites on the Mac:
- Go 1.24+ installed
- `real-esrgan-ncnn-vulkan` binary available (downloaded by the build script)

Command:
```bash
# Deploys code to Mac, builds App/DMG, and downloads artifacts back to ./build/macos
task macos:build
```

You can also run the service remotely on the Mac:
```bash
# Sync and run
task macos:start
# View logs
task macos:logs
# Stop service
task macos:stop
```

## API Usage

The API is fully documented with OpenAPI/Swagger.
Once the server is running, visit **http://localhost:8089/api/v1/docs** for the interactive UI.

For detailed endpoint documentation, examples, and parameter descriptions, see the **[API User Guide](docs/API_GUIDE.md)** or view the raw **[OpenAPI Specification](docs/openapi.yaml)**.

### Quick Examples

#### Upscale an Image (Async)

```bash
# 1. Submit Job
curl -X POST http://localhost:8089/api/v1/upscale \
  -F "image=@photo.jpg" \
  -F "scale=4" \
  -F "model_name=realesrgan-x4plus"

# Response: {"success": true, "job_id": "123...", ...}

# 2. Check Status
curl http://localhost:8089/api/v1/status/123...

# 3. Download (when status is "completed")
curl -O http://localhost:8089/api/v1/download/123...
```

#### List Available Models

```bash
curl http://localhost:8089/api/v1/models
```

## Client CLI

The project includes a CLI client for easy interaction with the API.

```bash
make build-client
./build/upscale-client -input image.jpg -output upscaled.png -scale 4
```

## Configuration

Configuration is managed via `config/config.yaml`. Key settings include:

*   **Server**: Port, timeouts, auth token.
*   **Upscaler**: GPU enable/disable, thread count, model path.
*   **Storage**: Upload/output directories, cleanup policies.
*   **Limits**: Concurrency, queue size.

For Docker, see `config/config.docker.yaml`.

## Directory Structure

*   `cmd/`: Entry points for server and client.
*   `internal/`: Core logic (API handlers, upscaler, storage).
*   `deployments/`: Docker configurations.
*   `models/`: AI models (downloaded via `make models`).
*   `bin/`: External binaries (ncnn-vulkan).
*   `data/`: Storage for uploads and output images.

## License

MIT License - Copyright (c) 2026 Michael Lechner