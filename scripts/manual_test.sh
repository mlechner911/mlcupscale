#!/bin/bash
set -e

FILE="$1"
SCALE="$2"
HOST="macstudio.fritz.box"
PORT="8089"
API_URL="http://$HOST:$PORT/api/v1"
MODEL_NAME="realesrgan-x4plus"

if [ -z "$FILE" ]; then
    echo "Usage: $0 <image_file> [scale]"
    exit 1
fi

if [ -z "$SCALE" ]; then
    SCALE="2"
fi

# Function to get current time in seconds with high precision
now() {
    date +%s.%N
}

# Function to calculate duration
calc() {
    python3 -c "print(f'{float($2) - float($1):.2f}')"
}

echo "=== Upscaling $FILE (x$SCALE) on $HOST ==="

# 1. Upload & Submit
echo "Submitting upscale job..."
T_START_UPLOAD=$(now)
RESPONSE=$(curl -s -F "image=@$FILE" -F "model_name=$MODEL_NAME" -F "scale=$SCALE" "$API_URL/upscale")
T_END_UPLOAD=$(now)
JOB_ID=$(echo "$RESPONSE" | grep -o '"job_id":"[^"]*"' | cut -d'"' -f4)

if [ -z "$JOB_ID" ]; then
    echo "Failed to submit job. Response:"
    echo "$RESPONSE"
    exit 1
fi

echo "Job ID: $JOB_ID"

# 2. Poll Status
echo "Polling status..."
T_START_PROCESS=$(now)
SERVER_DURATION=""

while true; do
    STATUS_RES=$(curl -s "$API_URL/status/$JOB_ID")
    STATE=$(echo "$STATUS_RES" | grep -o '"status":"[^"]*"' | cut -d'"' -f4)
    PROGRESS=$(echo "$STATUS_RES" | grep -o '"progress":[0-9]*' | cut -d':' -f2)

    if [ -z "$PROGRESS" ]; then PROGRESS="0"; fi

    echo -ne "Status: $STATE (Progress: ${PROGRESS}%)   \r"

    if [ "$STATE" == "completed" ]; then
        echo "" # New line
        T_END_PROCESS=$(now)

        # Extract server-side duration for report
        SERVER_DURATION=$(python3 -c "import sys, json; print(json.loads(sys.argv[1]).get('duration_seconds', '0'))" "$STATUS_RES")

        # Parse details for display
        python3 -c "import sys, json
try:
    data = json.loads(sys.argv[1])
    inp = data.get('input_size', {})
    out = data.get('output_size', {})
    print(f\"Original Size: {inp.get('width', '?')}x{inp.get('height', '?')}\")
    print(f\"Upscaled Size: {out.get('width', '?')}x{out.get('height', '?')}\")
except Exception as e:
    print(f\"Failed to parse details: {e}\")
" "$STATUS_RES"
        break
    elif [ "$STATE" == "failed" ] || [ "$STATE" == "cancelled" ]; then
        echo ""
        echo "Job failed/cancelled. Response:"
        echo "$STATUS_RES"
        exit 1
    fi
    sleep 1
done

# 3. Download
OUTPUT_FILE="${FILE%.*}.upscaled.${FILE##*.}"
echo "Downloading result to $OUTPUT_FILE..."
T_START_DOWNLOAD=$(now)
curl -s -o "$OUTPUT_FILE" "$API_URL/download/$JOB_ID"
T_END_DOWNLOAD=$(now)

# Calculate Durations
DMB_UPLOAD=$(calc $T_START_UPLOAD $T_END_UPLOAD)
DMB_PROCESS=$(calc $T_START_PROCESS $T_END_PROCESS)
DMB_DOWNLOAD=$(calc $T_START_DOWNLOAD $T_END_DOWNLOAD)
DMB_TOTAL=$(python3 -c "print(f'{float($DMB_UPLOAD) + float($DMB_PROCESS) + float($DMB_DOWNLOAD):.2f}')")

echo "✅ Done! Saved to $OUTPUT_FILE"
echo "---------------------------------------------------"
echo "Timing Report:"
echo "  Upload Time:      ${DMB_UPLOAD}s"
echo "  Client Wait Time: ${DMB_PROCESS}s (Server took ${SERVER_DURATION}s)"
echo "  Download Time:    ${DMB_DOWNLOAD}s"
echo "---------------------------------------------------"
echo "  Total Time:       ${DMB_TOTAL}s"
