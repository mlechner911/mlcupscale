#!/usr/bin/env python3
import argparse
import cv2
import numpy as np
import onnxruntime as ort
import os
import sys
import math
import time

def log_progress(current, total):
    """Prints progress percentage to stderr for the Go app to parse."""
    if total > 0:
        percent = (current / total) * 100.0
        sys.stderr.write(f"{percent:.2f}%\n")
        sys.stderr.flush()

def pre_process(img):
    """
    HWC (BGR) -> CHW (RGB) -> NCHW
    Normalize 0-255 -> 0-1
    """
    img = img.astype(np.float32) / 255.0
    # BGR to RGB
    img = img[:, :, [2, 1, 0]]
    # HWC to CHW
    img = np.transpose(img, (2, 0, 1))
    # Add batch dimension
    img = np.expand_dims(img, axis=0)
    return img

def post_process(output):
    """
    NCHW -> CHW -> HWC (RGB) -> HWC (BGR)
    Clip 0-1 -> 0-255
    """
    output = np.squeeze(output, axis=0)
    output = np.clip(output, 0, 1)
    output = np.transpose(output, (1, 2, 0))
    # RGB to BGR
    output = output[:, :, [2, 1, 0]]
    output = (output * 255.0).round().astype(np.uint8)
    return output

def process_tile(session, img_tile):
    """Run inference on a single tile."""
    input_name = session.get_inputs()[0].name
    img_input = pre_process(img_tile)
    result = session.run(None, {input_name: img_input})[0]
    return post_process(result)

def upscale_image(session, img, scale, tile_size, padding):
    h, w, c = img.shape

    # New dimensions
    out_h = h * scale
    out_w = w * scale
    output_img = np.zeros((out_h, out_w, c), dtype=np.uint8)

    # Calculate tiles
    # We tile the INPUT, then scale coordinates for output

    # If tile_size is 0 or larger than image, process whole
    if tile_size <= 0 or (w <= tile_size and h <= tile_size):
        log_progress(0, 100)
        res = process_tile(session, img)
        log_progress(100, 100)
        return res

    # Tiling logic
    x_steps = math.ceil(w / tile_size)
    y_steps = math.ceil(h / tile_size)
    total_tiles = x_steps * y_steps
    processed_tiles = 0

    for y in range(0, h, tile_size):
        for x in range(0, w, tile_size):
            # Input Coordinates
            # Add padding
            x_pad_left = padding if x > 0 else 0
            y_pad_top = padding if y > 0 else 0

            # We want 'tile_size' valid pixels, but we might reach edge
            x_valid = min(tile_size, w - x)
            y_valid = min(tile_size, h - y)

            x_pad_right = padding if (x + x_valid) < w else 0
            y_pad_bottom = padding if (y + y_valid) < h else 0

            # Crop input with padding
            crop_x_start = x - x_pad_left
            crop_y_start = y - y_pad_top
            crop_x_end = x + x_valid + x_pad_right
            crop_y_end = y + y_valid + y_pad_bottom

            tile = img[crop_y_start:crop_y_end, crop_x_start:crop_x_end]

            # Process
            out_tile = process_tile(session, tile)

            # Crop Output (remove padding)
            # Output padding is input_padding * scale
            out_pad_left = x_pad_left * scale
            out_pad_top = y_pad_top * scale
            out_pad_right = x_pad_right * scale
            out_pad_bottom = y_pad_bottom * scale

            out_h_valid = out_tile.shape[0] - out_pad_top - out_pad_bottom
            out_w_valid = out_tile.shape[1] - out_pad_left - out_pad_right

            # Paste into result
            out_y_start = y * scale
            out_x_start = x * scale

            # Safety check for dimensions
            paste_region = out_tile[out_pad_top : out_pad_top+out_h_valid,
                                    out_pad_left : out_pad_left+out_w_valid]

            output_img[out_y_start : out_y_start+out_h_valid,
                       out_x_start : out_x_start+out_w_valid] = paste_region

            processed_tiles += 1
            log_progress(processed_tiles, total_tiles)

    return output_img

def main():
    parser = argparse.ArgumentParser(description="Real-ESRGAN ONNX Upscaler")

    # Match ncnn-vulkan arguments
    parser.add_argument('-i', '--input', required=True, help='Input image path')
    parser.add_argument('-o', '--output', required=True, help='Output image path')
    parser.add_argument('-s', '--scale', type=int, default=4, help='Upscale scale (default 4)')
    parser.add_argument('-t', '--tile-size', type=int, default=0, help='Tile size (0=auto/none)')
    parser.add_argument('-m', '--model-path', default='models', help='Models directory')
    parser.add_argument('-n', '--model-name', default='realesrgan-x4plus', help='Model name (without .onnx)')
    parser.add_argument('-j', '--threads', type=str, default="1:2:2", help='Threads (ignored, handled by ORT)')
    parser.add_argument('-g', '--gpu-id', type=int, default=-1, help='GPU ID (-1=auto)')
    parser.add_argument('-f', '--format', type=str, default='png', help='Output format (png/jpg/webp)')

    # Internal tuning
    parser.add_argument('--padding', type=int, default=10, help='Tile padding pixel count')

    args = parser.parse_args()

    # Load Image
    if not os.path.exists(args.input):
        print(f"Error: Input file found: {args.input}", file=sys.stderr)
        sys.exit(1)

    img = cv2.imread(args.input, cv2.IMREAD_UNCHANGED)
    if img is None:
        print(f"Error: Failed to read image: {args.input}", file=sys.stderr)
        sys.exit(1)

    # Handle Alpha
    has_alpha = False
    alpha = None
    if img.ndim == 3 and img.shape[2] == 4:
        has_alpha = True
        alpha = img[:, :, 3]
        img = img[:, :, 0:3]

    # Load Model
    model_file = os.path.join(args.model_path, f"{args.model_name}.onnx")
    if not os.path.exists(model_file):
         print(f"Error: Model file not found: {model_file}", file=sys.stderr)
         sys.exit(1)

    # Session Options
    providers = []
    if args.gpu_id != -1:
        # Try CUDA if available
        if 'CUDAExecutionProvider' in ort.get_available_providers():
             providers.append(('CUDAExecutionProvider', {'device_id': args.gpu_id}))

    providers.append('CPUExecutionProvider')
    print(f"Loading model {model_file} with providers: {providers}", file=sys.stderr)

    try:
        sess = ort.InferenceSession(model_file, providers=providers)
    except Exception as e:
        print(f"Error loading model: {e}", file=sys.stderr)
        sys.exit(1)

    # Upscale
    start_time = time.time()
    upscaled_img = upscale_image(sess, img, args.scale, args.tile_size, args.padding)

    # Handle Alpha Resize
    if has_alpha:
        h, w = upscaled_img.shape[:2]
        upscaled_alpha = cv2.resize(alpha, (w, h), interpolation=cv2.INTER_CUBIC)
        upscaled_img = np.dstack((upscaled_img, upscaled_alpha))

    # Save
    os.makedirs(os.path.dirname(os.path.abspath(args.output)), exist_ok=True)
    success = cv2.imwrite(args.output, upscaled_img)

    if not success:
        print(f"Error: Failed to write output: {args.output}", file=sys.stderr)
        sys.exit(1)

    print(f"Done. Saved to {args.output} in {time.time()-start_time:.2f}s", file=sys.stderr)

if __name__ == "__main__":
    main()
