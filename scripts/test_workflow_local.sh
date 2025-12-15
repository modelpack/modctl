#!/bin/bash
# Test the build-top-models workflow locally
# This simulates the workflow without using GitHub Actions

set -e  # Exit on error

echo "======================================"
echo "Local Workflow Test"
echo "======================================"

# Configuration
LIMIT=${1:-3}
MAX_SIZE=${2:-5}
SORT_BY=${3:-downloads}
TEST_DIR="test-workflow-run"

echo ""
echo "Configuration:"
echo "  - Limit: $LIMIT models"
echo "  - Max Size: ${MAX_SIZE}GB"
echo "  - Sort By: $SORT_BY"
echo ""

# Step 1: Select models
echo "======================================"
echo "Step 1: Selecting compatible models..."
echo "======================================"

mkdir -p "$TEST_DIR"

python3 scripts/select_top_models.py \
  --limit "$LIMIT" \
  --max-size "$MAX_SIZE" \
  --sort-by "$SORT_BY" \
  --output "$TEST_DIR/models.json"

if [ ! -f "$TEST_DIR/models.json" ]; then
  echo "❌ Failed to generate models.json"
  exit 1
fi

echo ""
echo "Selected models:"
cat "$TEST_DIR/models.json" | jq -r '.[] | "  - \(.id) (\(.format), \(.size_gb)GB)"'

# Step 2: Check if modctl is available
echo ""
echo "======================================"
echo "Step 2: Checking modctl..."
echo "======================================"

if ! command -v modctl &> /dev/null; then
  echo "⚠️  modctl not found in PATH"
  if [ -f "./modctl" ]; then
    echo "✓ Using local modctl binary"
    MODCTL_CMD="./modctl"
  else
    echo "Building modctl..."
    go build -tags "static system_libgit2 enable_libgit2" -o modctl
    MODCTL_CMD="./modctl"
  fi
else
  echo "✓ modctl found in PATH"
  MODCTL_CMD="modctl"
fi

$MODCTL_CMD version

# Step 3: Test model download and Modelfile generation
echo ""
echo "======================================"
echo "Step 3: Testing first model..."
echo "======================================"

# Get first model from JSON
FIRST_MODEL=$(cat "$TEST_DIR/models.json" | jq -r '.[0]')
MODEL_ID=$(echo "$FIRST_MODEL" | jq -r '.id')
MODEL_FAMILY=$(echo "$FIRST_MODEL" | jq -r '.family')
MODEL_FORMAT=$(echo "$FIRST_MODEL" | jq -r '.format')
MODEL_PARAM_SIZE=$(echo "$FIRST_MODEL" | jq -r '.param_size')

echo "Testing with: $MODEL_ID"
echo "  Family: $MODEL_FAMILY"
echo "  Format: $MODEL_FORMAT"
echo "  Params: $MODEL_PARAM_SIZE"

MODEL_DIR="$TEST_DIR/model-files"
mkdir -p "$MODEL_DIR"

# Step 4: Download model (if not cached)
echo ""
echo "======================================"
echo "Step 4: Downloading model..."
echo "======================================"

if [ -d "$MODEL_DIR/config.json" ]; then
  echo "Model already downloaded, skipping..."
else
  echo "Downloading $MODEL_ID..."
  python3 << EOF
from huggingface_hub import snapshot_download
import os

model_id = "$MODEL_ID"
model_dir = "$MODEL_DIR"

print(f"Downloading {model_id} to {model_dir}...")
snapshot_download(
    repo_id=model_id,
    local_dir=model_dir,
)
print("✓ Download complete")
EOF
fi

# Step 5: Generate Modelfile
echo ""
echo "======================================"
echo "Step 5: Generating Modelfile..."
echo "======================================"

cd "$MODEL_DIR"

$MODCTL_CMD modelfile generate \
  --arch transformer \
  --family "$MODEL_FAMILY" \
  --format "$MODEL_FORMAT" \
  --param-size "$MODEL_PARAM_SIZE" \
  --overwrite \
  .

if [ ! -f "Modelfile" ]; then
  echo "❌ Failed to generate Modelfile"
  exit 1
fi

echo ""
echo "Generated Modelfile:"
echo "======================================"
cat Modelfile
echo "======================================"

cd - > /dev/null

# Step 6: Build (optional - can be slow)
echo ""
echo "======================================"
echo "Step 6: Build test (optional)"
echo "======================================"

read -p "Do you want to test building the model? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  cd "$MODEL_DIR"

  # Use local registry for testing
  IMAGE_NAME=$(echo "$MODEL_ID" | tr '[:upper:]' '[:lower:]' | tr '/' '-')
  IMAGE_TAG="localhost:5000/${IMAGE_NAME}:test"

  echo "Building as: $IMAGE_TAG"
  echo "Note: This will create local artifacts only (no push)"

  $MODCTL_CMD build -f Modelfile \
    -t "$IMAGE_TAG" \
    --log-level debug \
    .

  echo "✓ Build successful!"
  cd - > /dev/null
else
  echo "Skipping build test"
fi

# Summary
echo ""
echo "======================================"
echo "Test Summary"
echo "======================================"
echo "✓ Model selection: Success"
echo "✓ Model download: Success"
echo "✓ Modelfile generation: Success"
echo ""
echo "Test artifacts saved in: $TEST_DIR"
echo ""
echo "To clean up: rm -rf $TEST_DIR"
