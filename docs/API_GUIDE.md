# API User Guide: Image Upscaling

This guide provides step-by-step instructions for integrating with the Image Upscale Service API.

## Workflow Overview

The upscaling process is **asynchronous**:
1.  **Submit** an image to the queue.
2.  **Poll** the status to check progress.
3.  **Download** the result once complete.

> **Important Data Policy:** 
> *   Images are **deleted immediately** after you download them.
> *   If not downloaded, images are **automatically deleted after 15 minutes**.

---

## Step 1: Submit a Job

Send a `POST` request to `/api/v1/upscale` with your image file.

**Endpoint:** `POST /api/v1/upscale`
**Content-Type:** `multipart/form-data`

| Parameter | Type | Required | Description |
| :--- | :--- | :--- | :--- |
| `image` | File | Yes | The image file (PNG, JPG, WEBP). |
| `scale` | Int | No | Upscaling factor: `2`, `3`, or `4` (default: `4`). |

**Example Request:**
```bash
curl -X POST http://localhost:8089/api/v1/upscale \
  -F "image=@my_photo.jpg" \
  -F "scale=4"
```

**Example Response (202 Accepted):**
```json
{
  "success": true,
  "job_id": "1769781953720134401",
  "status_url": "/api/v1/status/1769781953720134401"
}
```

---

## Step 2: Check Status

Periodically check the status of your job using the `status_url` returned in Step 1.

**Endpoint:** `GET /api/v1/status/{job_id}`

**Example Request:**
```bash
curl http://localhost:8089/api/v1/status/1769781953720134401
```

**Response (Processing):**
```json
{
  "job_id": "1769781953720134401",
  "status": "processing",
  "progress": 45
}
```

**Response (Completed):**
```json
{
  "job_id": "1769781953720134401",
  "status": "completed",
  "progress": 100,
  "download_url": "/api/v1/download/1769781953720134401",
  "duration_seconds": 32.5,
  "file_size_bytes": 122013
}
```

---

## Step 3: Download Result

Once the status is `completed`, use the `download_url` to retrieve your upscaled image.

**Endpoint:** `GET /api/v1/download/{job_id}`

**Example Request:**
```bash
curl -O http://localhost:8089/api/v1/download/1769781953720134401
```

> **Note:** The file will be deleted from the server immediately after this request completes.

---

## Error Handling

If a job fails, the status response will look like this:

```json
{
  "job_id": "1769781953720134401",
  "status": "failed",
  "error": "upscale failed: model not found"
}
```
