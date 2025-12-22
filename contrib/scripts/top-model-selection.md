# Model Selection Scripts

This directory contains scripts for selecting and filtering HuggingFace models compatible with modctl.

## select-top-models.py

Python script that fetches top models from HuggingFace Hub and filters them based on modctl compatibility criteria.

### Compatibility Criteria

The script filters models based on:

1. **Has config.json** - Required for auto-detection of model metadata
2. **Supported formats** - Must have files in formats like:
   - `safetensors` (preferred)
   - `gguf`
   - `bin` (PyTorch)
   - `pt`, `pth` (PyTorch)
   - `onnx`
3. **Size limit** - Configurable maximum size (default: 20GB)
4. **Metadata** - Attempts to extract:
   - Model family (llama, qwen, gpt2, etc.)
   - Parameter size (0.5B, 7B, etc.)
   - Format type

### Installation

```bash
pip install -r requirements.txt
```

### Usage

Basic usage (fetch top 10 models by downloads):

```bash
python contrib/scripts/select-top-models.py
```

#### Options

```bash
python contrib/scripts/select-top-models.py \
  --limit 10 \              # Number of models to select (default: 10)
  --max-size 20.0 \         # Maximum model size in GB (default: 20.0)
  --sort-by downloads \     # Sort by: downloads, likes, trending (default: downloads)
  --task text-generation \  # Task filter (default: text-generation)
  --output models.json      # Output file (default: stdout)
```

#### Examples

Get top 5 small models (< 5GB):

```bash
python contrib/scriptsselect-top-models.py --limit 5 --max-size 5
```

Get most liked models:

```bash
python contrib/scripts/select-top-models.py --limit 10 --sort-by likes
```

Save to file:

```bash
python contrib/scripts/select-top-models.py --limit 20 --output top_models.json
```

### Output Format

The script outputs JSON with model metadata:

```json
[
  {
    "id": "Qwen/Qwen3-0.6B",
    "family": "qwen3",
    "arch": "transformer",
    "format": "safetensors",
    "param_size": "0.6B",
    "size_gb": 1.41,
    "downloads": 7509488,
    "likes": 867
  }
]
```

### Authentication

Some models require HuggingFace authentication. Set the `HF_TOKEN` environment variable:

```bash
export HF_TOKEN="your_huggingface_token"
python contrib/scripts/select-top-models.py
```

Or use `huggingface-cli`:

```bash
huggingface-cli login
python contrib/scripts/select-top-models.py
```

## GitHub Workflow Integration

The `build-top-models.yml` workflow uses this script to automatically:

1. Select top models from HuggingFace
2. Build them using modctl
3. Push to GitHub Container Registry

### Manual Trigger

You can manually trigger the workflow from GitHub Actions tab with custom parameters:

- **limit**: Number of models to build (default: 10)
- **max_size**: Maximum model size in GB (default: 20)
- **sort_by**: Sort criteria - downloads, likes, or trending

### Scheduled Runs

The workflow runs automatically every Sunday at 00:00 UTC.

### Required Secrets

The workflow requires these GitHub secrets:

- `HF_TOKEN` - HuggingFace API token (for downloading models)
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions
