#!/bin/bash
set -e

# Default to Mac address if not provided
TARGET_HOST="${1:-localhost}"
PORT="${2:-8089}"
API_URL="http://$TARGET_HOST:$PORT/api/v1"
IMAGE_FILE="/tmp/test_tiny.png"

# Generate a tiny 1x1 PNG (Red pixel)
# 1x1 PNG Base64
echo "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKwMTQAAAABJRU5ErkJggg==" | base64 -d > "$IMAGE_FILE"

echo "=== Verifying Upscale Service at $API_URL ==="
echo "Generated test image at $IMAGE_FILE"

# check health first
echo "Checking health..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/health")
if [ "$HTTP_CODE" != "200" ]; then
    echo "Health check failed (HTTP $HTTP_CODE). Is the service running?"
    exit 1
fi
echo "Health OK."

# Submit Job
echo "Submitting upscale job..."
RESPONSE=$(curl -s -F "image=@$IMAGE_FILE" -F "model_name=realesrgan-x4plus" "$API_URL/upscale")
# Simple grep parsing to avoid jq dependency
JOB_ID=$(echo "$RESPONSE" | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)
ERROR_MSG=$(echo "$RESPONSE" | grep -o '"error":"[^"]*"' | cut -d'"' -f4)

if [ -z "$JOB_ID" ]; then
    echo "Failed to submit job."
    echo "Response: $RESPONSE"
    exit 1
fi

echo "Job submitted. ID: $JOB_ID"

# Poll Status
echo "Waiting for completion..."
MAX_RETRIES=30
COUNT=0

while [ $COUNT -lt $MAX_RETRIES ]; do
    STATUS_RES=$(curl -s "$API_URL/status/$JOB_ID")

    # Extract status and error fields
    STATE=$(echo "$STATUS_RES" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    # Extract error message if present (simple regex, might miss complex json escaping but good enough for basic errors)
    ERR_VAL=$(echo "$STATUS_RES" | sed -n 's/.*"error":"\([^"]*\)".*/\1/p')

    echo "[$COUNT] Status: $STATE"

    if [ "$STATE" == "completed" ]; then
        echo "✅ SUCCESS: Image upscaled successfully."
        # Optional: check download url access
        exit 0
    elif [ "$STATE" == "failed" ]; then
        echo "❌ FAILED: Job failed."
        echo "Error details: $ERR_VAL"
        exit 1
    elif [ "$STATE" == "cancelled" ]; then
        echo "⚠️ CANCELLED: Job was cancelled."
        exit 1
    fi

    sleep 2
    COUNT=$((COUNT+1))
done

echo "❌ TIMEOUT: Job did not complete in time."
exit 1
