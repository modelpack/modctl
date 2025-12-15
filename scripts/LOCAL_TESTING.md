# Local Testing Guide

This guide shows you how to test the `build-top-models` workflow locally without triggering it on GitHub.

## Quick Start (Recommended)

### Option 1: Dry Run Validation (Fastest - 30 seconds)

Validates the workflow logic without downloading models:

```bash
python3 scripts/validate_workflow.py
```

This will:
- ✓ Check all prerequisites
- ✓ Run model selection
- ✓ Validate model metadata
- ✓ Show what commands would be executed
- ✓ Estimate storage requirements

**Use this for:** Quick validation before pushing workflow changes

---

### Option 2: Full Simulation (Slow - 5-15 minutes)

Simulates the entire workflow including model download and Modelfile generation:

```bash
./scripts/test_workflow_local.sh
```

With custom parameters:

```bash
./scripts/test_workflow_local.sh <limit> <max_size_gb> <sort_by>

# Examples:
./scripts/test_workflow_local.sh 3 5 downloads      # 3 models, max 5GB
./scripts/test_workflow_local.sh 1 2 likes          # 1 model, max 2GB
```

This will:
1. Select compatible models
2. Check/build modctl binary
3. Download the first model from HuggingFace
4. Generate Modelfile
5. Optionally test the build (you'll be prompted)

**Use this for:** Testing the complete flow before first GitHub run

---

### Option 3: Using `act` (GitHub Actions locally)

[`act`](https://github.com/nektos/act) runs GitHub Actions workflows locally using Docker.

#### Installation

**macOS:**
```bash
brew install act
```

**Linux:**
```bash
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash
```

**Windows:**
```bash
choco install act-cli
```

#### Basic Usage

List available workflows:
```bash
act -l
```

Run the workflow (dry run):
```bash
act workflow_dispatch --dry-run
```

Run the workflow (select-models job only):
```bash
act workflow_dispatch -j select-models
```

Run with secrets:
```bash
# Create .secrets file:
# HF_TOKEN=your_token_here

act workflow_dispatch --secret-file .secrets
```

Run with custom inputs:
```bash
act workflow_dispatch \
  -j select-models \
  --input limit=3 \
  --input max_size=5 \
  --input sort_by=downloads
```

#### Limitations

- `act` requires Docker
- May not support all GitHub Actions features
- Large models will consume significant disk space
- Build jobs may be very slow in Docker

**Use this for:** Testing workflow YAML syntax and job dependencies

---

## Step-by-Step Manual Testing

If you want fine-grained control, test each step individually:

### Step 1: Test Model Selection

```bash
python3 scripts/select_top_models.py \
  --limit 3 \
  --max-size 5 \
  --output test-models.json

# View results
cat test-models.json | jq '.'
```

### Step 2: Verify modctl is Available

```bash
# Check if modctl exists
which modctl || echo "Not in PATH"

# Or use local binary
./modctl version
```

### Step 3: Download a Test Model

```bash
# Install huggingface_hub
pip install huggingface_hub

# Download a small model
python3 << 'EOF'
from huggingface_hub import snapshot_download

snapshot_download(
    repo_id="Qwen/Qwen3-0.6B",
    local_dir="./test-model"
)
EOF
```

### Step 4: Generate Modelfile

```bash
cd test-model

../modctl modelfile generate \
  --arch transformer \
  --family qwen3 \
  --format safetensors \
  --param-size 0.6B \
  .

cat Modelfile
```

### Step 5: Test Build (Optional)

```bash
# Build locally (no push)
../modctl build \
  -f Modelfile \
  -t test-model:local \
  --log-level debug \
  .
```

### Step 6: Cleanup

```bash
cd ..
rm -rf test-model test-models.json
```

---

## Testing Specific Workflow Components

### Test Only Model Selection Logic

```bash
# Test different sort criteria
python3 scripts/select_top_models.py --sort-by likes --limit 5
python3 scripts/select_top_models.py --sort-by trending --limit 5

# Test size filtering
python3 scripts/select_top_models.py --max-size 1  # Very small models
python3 scripts/select_top_models.py --max-size 50 # Larger models

# Test different tasks
python3 scripts/select_top_models.py --task image-classification
```

### Test JSON Output Parsing

```bash
# Generate JSON
python3 scripts/select_top_models.py --limit 3 > models.json

# Test jq parsing (same as workflow uses)
cat models.json | jq -c '.'
cat models.json | jq -r '.[] | "\(.id) (\(.format), \(.size_gb)GB)"'

# Test matrix conversion (how GitHub uses it)
echo "models=$(cat models.json | jq -c)" > /dev/null
```

### Test Modelfile Generation Parameters

```bash
cd test-model

# Test with minimal parameters
modctl modelfile generate --arch transformer .

# Test with all parameters
modctl modelfile generate \
  --arch transformer \
  --family qwen3 \
  --format safetensors \
  --param-size 0.6B \
  --precision bfloat16 \
  .

# Test overwrite
modctl modelfile generate --arch transformer --overwrite .
```

---

## Troubleshooting Local Tests

### Python Import Errors

```bash
# Ensure dependencies are installed
pip install -r scripts/requirements.txt

# Or use virtual environment
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
pip install -r scripts/requirements.txt
```

### HuggingFace Authentication

```bash
# Option 1: Environment variable
export HF_TOKEN="your_token_here"

# Option 2: Login via CLI
pip install huggingface_hub
huggingface-cli login

# Verify
python3 -c "from huggingface_hub import HfApi; print('Auth OK')"
```

### modctl Not Found

```bash
# Build modctl if not present
go build -tags "static system_libgit2 enable_libgit2" -o modctl

# Or use from PATH
export PATH=$PATH:/path/to/modctl
```

### Disk Space Issues

```bash
# Check available space
df -h .

# Clear HuggingFace cache if needed
rm -rf ~/.cache/huggingface/hub/

# Clear test artifacts
rm -rf test-workflow-run/
```

### Out of Memory

```bash
# For large models, monitor memory usage
top  # or htop

# Reduce model size limit
python3 scripts/select_top_models.py --max-size 2
```

---

## CI/CD Testing Best Practices

1. **Always run validation first**
   ```bash
   python3 scripts/validate_workflow.py
   ```

2. **Test with small models initially**
   ```bash
   ./scripts/test_workflow_local.sh 1 2 downloads
   ```

3. **Verify JSON output format**
   ```bash
   python3 scripts/select_top_models.py --limit 3 | jq '.'
   ```

4. **Test on same OS as GitHub runners**
   - GitHub uses `ubuntu-latest` (Ubuntu 22.04)
   - Use Docker or VM if testing on different OS

5. **Check workflow syntax**
   ```bash
   # Install actionlint
   brew install actionlint  # or download from GitHub

   # Lint workflow file
   actionlint .github/workflows/build-top-models.yml
   ```

---

## What Each Test Method Covers

| Test Method | Speed | Model Download | Build Test | GitHub Syntax | Coverage |
|-------------|-------|----------------|------------|---------------|----------|
| `validate_workflow.py` | ⚡ Fast (30s) | ❌ No | ❌ No | ❌ No | Logic only |
| `test_workflow_local.sh` | 🐢 Slow (5-15m) | ✅ Yes | 🟡 Optional | ❌ No | Full workflow |
| `act` | 🐌 Very slow | ✅ Yes | ✅ Yes | ✅ Yes | Everything |
| Manual steps | 🎯 Variable | 🟡 Optional | 🟡 Optional | ❌ No | Specific parts |

---

## Recommended Testing Flow

Before first GitHub run:

```bash
# 1. Quick validation
python3 scripts/validate_workflow.py

# 2. Full simulation with 1 small model
./scripts/test_workflow_local.sh 1 2 downloads

# 3. Check workflow syntax
actionlint .github/workflows/build-top-models.yml  # if installed

# 4. Push and test with manual trigger on GitHub
# Use: limit=1, max_size=2
```

For workflow changes:

```bash
# 1. Validate logic
python3 scripts/validate_workflow.py

# 2. Lint YAML
actionlint .github/workflows/build-top-models.yml

# 3. Test locally if needed
./scripts/test_workflow_local.sh 1 2
```

For script changes:

```bash
# Test with different parameters
python3 scripts/select_top_models.py --limit 5 --max-size 5
python3 scripts/select_top_models.py --limit 10 --sort-by likes
python3 scripts/select_top_models.py --limit 3 --task image-classification

# Validate output
python3 scripts/validate_workflow.py
```
