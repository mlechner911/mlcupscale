#!/bin/bash
set -e

BASE_URL="http://localhost:8089/api/v1"
TEST_IMAGE="test/testdata/sample_images/test.png"
OUTPUT_IMAGE="test_output.png"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[TEST]${NC} $1"
}

fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    exit 1
}

# Ensure dependencies
command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v jq >/dev/null 2>&1 || fail "jq is required"

# 1. Test Health
log "Testing Health Endpoint..."
HEALTH_STATUS=$(curl -s "$BASE_URL/health" | jq -r .status)
VERSION=$(curl -s "$BASE_URL/health" | jq -r .version)

if [ "$HEALTH_STATUS" != "ok" ]; then
    fail "Health check failed: status=$HEALTH_STATUS"
fi
if [ "$VERSION" != "1.0.0" ]; then
    fail "Version check failed: version=$VERSION"
fi
log "Health check passed (v$VERSION)"

# 2. Test Models List
log "Testing Models List..."
MODELS_COUNT=$(curl -s "$BASE_URL/models" | jq '.models | length')
if [ "$MODELS_COUNT" -eq 0 ]; then
    fail "No models found"
fi
log "Found $MODELS_COUNT models"

# 3. Test Models Filtering (Scale=2)
log "Testing Models Filtering (scale=2)..."
FILTERED_COUNT=$(curl -s "$BASE_URL/models?scale=2" | jq '.models | length')
# We know 'realesrgan-x4plus-anime' supports only 4x, so count should be less than total if that model exists
# Or simply check that all returned models support scale 2
SUPPORTED=$(curl -s "$BASE_URL/models?scale=2" | jq '.models[].supported_scales | contains([2])' | grep -v "true" | wc -l)
if [ "$SUPPORTED" -ne 0 ]; then
    fail "Filtered models contain unsupported scales"
fi
log "Filtering passed ($FILTERED_COUNT models support 2x)"

# 4. Test Upscale
log "Testing Upscale (4x)..."
if [ ! -f "$TEST_IMAGE" ]; then
    # Create test image if not exists
    log "Creating test image..."
    mkdir -p $(dirname "$TEST_IMAGE")
    convert -size 50x50 gradient:blue-red "$TEST_IMAGE" || python3 -c "from PIL import Image; Image.new('RGB', (50, 50), color = 'red').save('$TEST_IMAGE')"
fi

RESPONSE=$(curl -s -X POST "$BASE_URL/upscale" \
  -F "image=@$TEST_IMAGE" \
  -F "scale=4" \
  -F "model_name=realesrgan-x4plus")

SUCCESS=$(echo "$RESPONSE" | jq -r .success)
JOB_ID=$(echo "$RESPONSE" | jq -r .job_id)

if [ "$SUCCESS" != "true" ]; then
    ERROR=$(echo "$RESPONSE" | jq -r .error)
    fail "Upscale failed: $ERROR"
fi
log "Upscale job submitted (ID: $JOB_ID)"

# 5. Wait for download
log "Waiting for result..."
DOWNLOAD_URL=$(echo "$RESPONSE" | jq -r .download_url)
FULL_DOWNLOAD_URL="http://localhost:8089$DOWNLOAD_URL"

# Simple retry loop to check if file is ready (though response implies completion in sync mode)
# Our current implementation waits for completion before returning, so it should be ready.
curl -s -o "$OUTPUT_IMAGE" "$FULL_DOWNLOAD_URL"

if [ ! -f "$OUTPUT_IMAGE" ]; then
    fail "Download failed"
fi

FILE_SIZE=$(stat -c%s "$OUTPUT_IMAGE")
if [ "$FILE_SIZE" -lt 100 ]; then
    fail "Output image too small ($FILE_SIZE bytes)"
fi

log "Upscale successful! Output saved to $OUTPUT_IMAGE"

# Cleanup
rm -f "$OUTPUT_IMAGE"

log "All tests passed!"
