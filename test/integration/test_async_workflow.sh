#!/bin/bash
set -e

# Configuration
API_URL="http://localhost:8089/api/v1"
SCALE=${1:-3} # Default scale is 3, can be overridden by first argument
TEST_DIR="test/testdata/generated"
OUTPUT_DIR="test/results"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Starting Async Upscale Test (Scale: ${SCALE}x) ===${NC}"

# Create directories
mkdir -p "$TEST_DIR" "$OUTPUT_DIR"

# Generate test images if they don't exist
echo "Generating test images..."
if [ ! -f "$TEST_DIR/test_512.png" ]; then
    convert -size 512x512 xc:blue -fill white -draw "circle 256,256 128,128" "$TEST_DIR/test_512.png"
fi
if [ ! -f "$TEST_DIR/test_768.png" ]; then
    convert -size 768x768 xc:green -fill yellow -draw "rectangle 200,200 568,568" "$TEST_DIR/test_768.png"
fi
if [ ! -f "$TEST_DIR/test_1024.png" ]; then
    convert -size 1024x1024 xc:red -fill black -draw "text 400,512 'Hello World'" "$TEST_DIR/test_1024.png"
fi

# Function to run test
run_test() {
    local input_file="$1"
    local filename=$(basename "$input_file")
    
    echo -e "\n${BLUE}Testing: $filename${NC}"
    
    # 1. Submit Job
    echo "Submitting job..."
    response=$(curl -s -X POST "$API_URL/upscale" \
        -F "image=@$input_file" \
        -F "scale=$SCALE")
    
    # Extract ID using grep/cut (simple JSON parsing)
    job_id=$(echo "$response" | grep -o '"job_id":"[^" ]*"' | cut -d'"' -f4)
    status_url=$(echo "$response" | grep -o '"status_url":"[^" ]*"' | cut -d'"' -f4)
    
    if [ -z "$job_id" ]; then
        echo "Failed to submit job. Response: $response"
        return 1
    fi
    
    echo "Job ID: $job_id"
    
    # 2. Poll Status
    echo -n "Processing: "
    while true; do
        status_resp=$(curl -s "http://localhost:8089$status_url")
        status=$(echo "$status_resp" | grep -o '"status":"[^" ]*"' | cut -d'"' -f4)
        progress=$(echo "$status_resp" | grep -o '"progress":[^,}]*' | cut -d':' -f2 | tr -d ' ')
        
        if [ "$status" = "completed" ]; then
            echo -e " ${GREEN}Done!${NC}"
            download_url=$(echo "$status_resp" | grep -o '"download_url":"[^" ]*"' | cut -d'"' -f4)
            break
        elif [ "$status" = "failed" ]; then
            echo "Failed!"
            echo "Response: $status_resp"
            return 1
        else
            if [ -z "$progress" ]; then progress="0"; fi
            echo -n "$progress% "
            sleep 1
        fi
    done
    
    # 3. Download Result
    echo "Downloading result..."
    curl -s -o "$OUTPUT_DIR/upscaled_$filename" "http://localhost:8089$download_url"
    
    # Verify
    if [ -f "$OUTPUT_DIR/upscaled_$filename" ]; then
        echo -e "${GREEN}Success! Saved to $OUTPUT_DIR/upscaled_$filename${NC}"
    else
        echo "Download failed."
        return 1
    fi
}

# Run tests
run_test "$TEST_DIR/test_512.png"
run_test "$TEST_DIR/test_768.png"
run_test "$TEST_DIR/test_1024.png"

echo -e "\n${GREEN}=== All tests completed ===${NC}"