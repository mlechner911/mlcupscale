# Quick Start Guide

This guide will help you get the **Image Upscale Service** up and running in minutes using Docker.

## Prerequisites

*   **Docker** (and Docker Compose Plugin) installed.
*   **Make** (optional, but recommended).
*   **Git**.

## 1. Setup

Clone the repository and download the required AI models:

```bash
# Clone repo (if not already done)
git clone <repository-url>
cd upscale-service

# Download models
make models
```

## 2. Start the Service

The easiest way to run the service is with Docker. This handles all dependencies (ncnn, vulkan, etc.).

```bash
make docker-run
```

*   The service will start on port **8089**.
*   API endpoint: `http://localhost:8089/api/v1`

## 3. Test the Service

You can verify the service is running by checking the health endpoint:

```bash
curl http://localhost:8089/api/v1/health
# Output: {"status":"ok","version":"1.0.0",...}
```

Or run the automated integration tests:

```bash
make test-integration
```

## 4. Upscale an Image

### Using the CLI Client

We provide a simple Go-based CLI client. Build it first:

```bash
make client-build
```

Then run it:

```bash
./build/upscale-client \
  -server http://localhost:8089 \
  -input path/to/your/image.jpg \
  -output result.png \
  -scale 4
```

### Using cURL

You can also use standard HTTP requests:

```bash
curl -X POST http://localhost:8089/api/v1/upscale \
  -F "image=@my-photo.jpg" \
  -F "scale=4" \
  -o result.json
```

Then download the file using the URL provided in the JSON response.

## 5. API Documentation

For full API details, see the [OpenAPI Specification](openapi.yaml).

```