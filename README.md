# Image Upscale Service

A high-performance, containerized REST API for AI-based image upscaling using Real-ESRGAN.

## Features

*   **AI Upscaling**: Uses Real-ESRGAN (ncnn-vulkan) for 2x, 3x, and 4x upscaling.
*   **Performance**: Optimized for GPU (Vulkan) and CPU execution.
*   **API**: Clean REST API with job status tracking.
*   **Deployment**: Docker-ready with a multi-stage build.

## Documentation

*   [**Quick Start Guide**](docs/QUICKSTART.md) - Get up and running in minutes.
*   [**API Reference**](docs/openapi.yaml) - Full OpenAPI 3.0 specification.
*   [**Project Plan**](projectplan.md) - Detailed architecture and design.

## Usage

### Start with Docker

```bash
make models
make docker-run
```

The service will be available at `http://localhost:8089`.

### Build Client

```bash
make client-build
./build/upscale-client -help
```

## License

MIT License - Copyright (c) 2026 Michael Lechner
