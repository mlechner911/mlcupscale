# Image Upscale Service

> **[mlcgo.eu](https://mlcgo.eu)** — tools, libraries and manuals · [Product page](https://mlcgo.eu/products/mlcupscale/)


A high-performance, containerized REST API and standalone CLI for AI-based image upscaling using Real-ESRGAN (ncnn-vulkan).

![Go Version](https://img.shields.io/badge/Go-1.24-blue)
![License](https://img.shields.io/badge/License-MIT-green)
![Status](https://img.shields.io/badge/Status-Beta-orange)

## Why?

I wrote this "upscale service" to handle **huge images** (e.g., 10,000x10,000 pixels) that require significant compute power to upscale even further.

While this project is based on the excellent [Real-ESRGAN-ncnn-vulkan](https://github.com/xinntao/Real-ESRGAN-ncnn-vulkan) by xinntao, it modernizes the integration by:
1.  **Building from Source**: The core engine is compiled from the latest C++ source in our CI to ensure security, stability, and compatibility with modern hardware.
2.  **Go Wrapper/API**: Providing a robust, asynchronous Go-based REST API for server-side processing.
3.  **Standalone CLI**: A single-executable tool that bundles the engine and models for easy use on Windows, macOS, and Linux.

## Features

*   **AI Upscaling**: High-quality 2x, 3x, and 4x image upscaling.
*   **Models**: Includes `realesrgan-x4plus`, `realesrgan-x4plus-anime`, and `realesr-animevideov3`.
*   **Performance**: Optimized for GPU (Vulkan) with CPU fallback.
*   **Standalone CLI**: Use `./mlcupscale-cli` directly without a server.
*   **Asynchronous API**: Modern REST API with job tracking and swagger documentation.
*   **Production Ready**: Docker support, health checks, metrics, and rate limiting.

## Prerequisites

*   **Linux/macOS** (Windows requires WSL2 for server, or native for CLI)
*   **Go 1.24+** (for building locally)
*   **Task** (recommended) or **Make**
*   **Docker & Docker Compose** (optional)
*   **Vulkan Driver** (required for GPU acceleration)

## Getting Started

### 1. Standalone CLI (Quickest)

You can download the standalone CLI from the [GitHub Releases](https://github.com/mlechner911/mlcupscale/releases) page. It contains everything needed in one file.

```bash
# Build it yourself
task build-cli
./build/mlcupscale-cli-linux -i photo.jpg -o result.png
```

### 2. Run the Service (Server)

Build and start the REST API server:

```bash
# Setup: Download/Build engines and models
task binaries
task models

# Run
task run
```

The server will start at `http://localhost:8089`. Visit `http://localhost:8089/api/v1/docs` for the Swagger UI.

### 3. Run with Docker

```bash
make docker-run
```

## Performance & Testing

The service uses a specialized tiling strategy (default 512px) to handle massive images that typically fail on consumer hardware due to memory constraints (OOM).

| Test Case | Upscale Factor | Input Resolution | Output Resolution |
| :--- | :--- | :--- | :--- |
| **Normal** | 4x | 1920x1080 (2K) | 7680x4320 (8K) |
| **Extreme** | 4x | 3840x2160 (4K) | 15360x8640 (16K) |

## API Usage

### Submission (Async)

```bash
curl -X POST http://localhost:8089/api/v1/upscale \
  -F "image=@photo.jpg" \
  -F "scale=4"
```

Refer to **[API User Guide](docs/API_GUIDE.md)** for more details.

## License

MIT License - Copyright (c) 2026 Michael Lechner
Real-ESRGAN core by xinntao.
