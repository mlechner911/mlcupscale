# High-Performance AI Image Upscaling

MLCUpscale is a high-performance REST API designed for one specific task: upscaling massive images using state-of-the-art AI models. While many tools struggle with images over a few megapixels, MLCUpscale is built to handle files as large as 10,000x10,000 pixels by utilizing advanced tiling strategies and GPU acceleration.

## Beyond Consumer Hardware Limits

By offloading the compute-intensive upscaling process to a dedicated server, clients can process professional-grade imagery without local hardware constraints. MLCUpscale utilizes Vulkan-based hardware acceleration, making it compatible with a wide range of GPUs on both Linux and macOS (including Apple Silicon).

## Asynchronous Workflow

1. **Submit**: Send a POST request with your image and desired scale factor (2x, 3x, or 4x).
2. **Track**: Poll the status endpoint to monitor progress in real-time.
3. **Download**: Once completed, retrieve the high-resolution result.

## API Quickstart

Submit a new upscaling job:

```bash
curl -X POST http://api.mlcgo.eu/v1/upscale \
  -F "image=@photo.jpg" \
  -F "scale=4" \
  -F "model_name=realesrgan-x4plus"
```

Check the status:

```bash
curl http://api.mlcgo.eu/v1/status/YOUR_JOB_ID
```

Download the result:

```bash
curl -O http://api.mlcgo.eu/v1/download/YOUR_JOB_ID
```
