# API User Guide: Image Upscale Service

This comprehensive guide details the REST API endpoints for the Image Upscale Service. The API follows RESTful principles and returns JSON responses.

**OpenAPI Specification:** [openapi.yaml](openapi.yaml)  
**Swagger UI:** Available at `/api/v1/docs` when the server is running.

---

## Base URL

By default, the API is served at:
```
http://localhost:8089/api/v1
```

## Endpoints Overview

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| **POST** | `/upscale` | Submit a new image upscaling job. |
| **GET** | `/status/{job_id}` | Check the status and progress of a job. |
| **GET** | `/download/{job_id}` | Download the processed image (deletes file after). |
| **POST** | `/cancel/{job_id}` | Cancel a queued or running job. |
| **GET** | `/models` | List available AI models. |
| **GET** | `/health` | Check service health and version. |

---

## 1. Submit a Job
**`POST /upscale`**

Upload an image to be processed. This is an **asynchronous** operation. The response contains a `job_id` needed to track progress.

### Request Parameters (Multipart Form-Data)

| Parameter | Type | Required | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `image` | File | **Yes** | - | The image file to upscale. Supports PNG, JPG, WEBP. |
| `scale` | Integer | No | `4` | Upscaling factor. Allowed values: `2`, `3`, `4`. |
| `model_name`| String | No | `realesrgan-x4plus` | Specific model to use. See `/models` for options. |
| `format` | String | No | (Original) | Target output format: `png`, `jpg`, or `webp`. |
| `tile_size` | Integer | No | `0` (Auto) | Tile size for splitting large images to save VRAM. Use `400` or lower for low-VRAM GPUs. |

### Example Request
```bash
curl -X POST http://localhost:8089/api/v1/upscale \
  -F "image=@photo.jpg" \
  -F "scale=4" \
  -F "format=png"
```

### Response (202 Accepted)
```json
{
  "success": true,
  "job_id": "1769781953720134401",
  "status_url": "/api/v1/status/1769781953720134401"
}
```

---

## 2. Check Job Status
**`GET /status/{job_id}`**

Poll this endpoint to check the progress of your job.

### Path Parameters
| Parameter | Type | Description |
| :--- | :--- | :--- |
| `job_id` | String | The ID returned by the `/upscale` endpoint. |

### Response States

**State: Processing**
```json
{
  "job_id": "1769781953720134401",
  "status": "processing",
  "progress": 45
}
```

**State: Completed**
```json
{
  "job_id": "1769781953720134401",
  "status": "completed",
  "progress": 100,
  "download_url": "/api/v1/download/1769781953720134401",
  "duration_seconds": 2.5,
  "input_size": { "width": 800, "height": 600 },
  "output_size": { "width": 3200, "height": 2400 },
  "file_size_bytes": 4501239
}
```

**State: Failed**
```json
{
  "job_id": "1769781953720134401",
  "status": "failed",
  "error": "image decode failed: invalid format"
}
```

---

## 3. Download Result
**`GET /download/{job_id}`**

Retrieve the final upscaled image.

> **Note:** This endpoint streams the file as binary content (`application/octet-stream`).
> **Important:** The file is **deleted from the server** immediately after the download completes to save space.

### Example Request
```bash
curl -OJ http://localhost:8089/api/v1/download/1769781953720134401
```

---

## 4. Helper Endpoints

### List Models
**`GET /models`**  
Returns a list of installed Real-ESRGAN models.

**Query Parameters:**
*   `scale` (optional): Filter models by supported scale (e.g., `?scale=3`).

**Response:**
```json
{
  "version": "1.0.0",
  "models": [
    {
      "name": "realesrgan-x4plus",
      "description": "General purpose, high quality",
      "supported_scales": [4]
    },
    {
      "name": "realesr-animevideov3",
      "description": "Optimized for anime/animation",
      "supported_scales": [2, 3, 4]
    }
  ]
}
```

### Cancel Job
**`POST /cancel/{job_id}`**  
Cancels a job if it is queued or currently processing.

### Health Check
**`GET /health`**  
Returns service status and version. Useful for readiness probes.

```json
{
  "status": "ok",
  "version": "1.0.0",
  "time": 1709223344
}
```
