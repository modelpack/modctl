#!/usr/bin/env python3
"""
Select top HuggingFace models compatible with modctl.

This script fetches popular models from HuggingFace Hub and filters them
based on modctl compatibility criteria:
1. Has config.json for auto-detection
2. Has model files in supported formats (safetensors, gguf, bin, pt)
3. Size is under a reasonable limit
4. Has necessary metadata for modelfile generation
"""

import json
import re
import sys
import argparse
from typing import List, Dict, Optional
from huggingface_hub import HfApi

# Try to import ModelFilter, fall back to dict if not available
try:
    from huggingface_hub import ModelFilter
except ImportError:
    ModelFilter = None


# Supported model file formats (based on pkg/modelfile/constants.go)
SUPPORTED_FORMATS = [
    "safetensors",
    "gguf",
    "bin",
    "pt",
    "pth",
    "onnx",
]

# Model families known to work well with modctl
KNOWN_FAMILIES = {
    "llama",
    "qwen",
    "qwen2",
    "qwen3",
    "mistral",
    "phi",
    "gpt2",
    "gpt_neo",
    "gpt_neox",
    "bloom",
    "opt",
    "falcon",
    "mpt",
    "stablelm",
}


def get_model_size_gb(model_info) -> Optional[float]:
    """Estimate model size in GB from model info."""
    try:
        total_size = 0
        if hasattr(model_info, 'siblings') and model_info.siblings:
            for file in model_info.siblings:
                if hasattr(file, 'size') and file.size:
                    total_size += file.size
        return total_size / (1024 ** 3)  # Convert to GB
    except Exception as e:
        print(f"Error: An error occurred in get_model_size_gb: {e}", file=sys.stderr)
        return None


def has_config_json(model_info) -> bool:
    """Check if model has config.json for auto-detection."""
    try:
        if hasattr(model_info, 'siblings') and model_info.siblings:
            return any(f.rfilename == "config.json" for f in model_info.siblings)
        return False
    except Exception as e:
        print(f"Error: An error occurred in has_config_json: {e}", file=sys.stderr)
        return None


def get_model_format(model_info) -> Optional[str]:
    """Detect model format from repository files."""
    try:
        if not hasattr(model_info, 'siblings') or not model_info.siblings:
            return None

        # Check for each supported format
        for file in model_info.siblings:
            filename = file.rfilename.lower()
            if filename.endswith('.safetensors'):
                return "safetensors"
            elif filename.endswith('.gguf'):
                return "gguf"
            elif filename.endswith('.onnx'):
                return "onnx"
            elif filename.endswith('.bin') and 'pytorch_model' in filename:
                return "bin"
            elif filename.endswith('.pt') or filename.endswith('.pth'):
                return "pt"

        return None
    except Exception as e:
        print(f"Error: An error occurred in get_model_format: {e}", file=sys.stderr)
        return None


def extract_param_size(model_id: str) -> Optional[str]:
    """Extract parameter size from model name or metadata."""
    # Common patterns: 7B, 8B, 13B, 0.5B, 1.1B, etc.
    patterns = [
        r'(\d+\.?\d*[BM])',  # 7B, 8B, 0.5B
        r'(\d+\.?\d*)b',     # 7b, 0.5b (lowercase)
    ]

    model_name = model_id.lower()
    for pattern in patterns:
        match = re.search(pattern, model_name)
        if match:
            size = match.group(1).upper()
            if not size.endswith('B') and not size.endswith('M'):
                size += 'B'
            return size

    return None


def detect_family(model_info, model_id: str) -> Optional[str]:
    """Detect model family from model info."""
    try:
        # Try to get from config
        if hasattr(model_info, 'config') and model_info.config:
            model_type = model_info.config.get('model_type')
            if model_type and model_type in KNOWN_FAMILIES:
                return model_type

        # Fallback to tags
        if hasattr(model_info, 'tags') and model_info.tags:
            for tag in model_info.tags:
                if tag in KNOWN_FAMILIES:
                    return tag

        # Last resort: parse from model name
        model_name_lower = model_id.lower()
        for family in KNOWN_FAMILIES:
            if family in model_name_lower:
                return family

        return None
    except Exception as e:
        print(f"Error: An error occurred in detect_family: {e}", file=sys.stderr)
        return None


def is_compatible_model(api: HfApi, model_id: str, max_size_gb: float = 20.0) -> tuple[bool, Optional[Dict]]:
    """
    Check if model is compatible with modctl.

    Returns:
        (is_compatible, model_metadata) tuple
    """
    # Get all model information
    try:
        model_info = api.model_info(model_id, files_metadata=True)
    except Exception as e:
        print(f"Skipping {model_id}: Could not fetch model info: {e}", file=sys.stderr)
        return False, None

    # Check for config.json
    if not has_config_json(model_info):
        print(f"Skipping {model_id}: No config.json", file=sys.stderr)
        return False, None

    # Check format
    format_type = get_model_format(model_info)
    if not format_type:
        print(f"Skipping {model_id}: No supported model format found", file=sys.stderr)
        return False, None

    # Check size
    size_gb = get_model_size_gb(model_info)
    if size_gb and size_gb > max_size_gb:
        print(f"Skipping {model_id}: Too large ({size_gb:.2f}GB > {max_size_gb}GB)", file=sys.stderr)
        return False, None

    # Detect family
    family = detect_family(model_info, model_id)

    # Extract param size
    param_size = extract_param_size(model_id)

    metadata = {
        "id": model_id,
        "family": family or "unknown",
        "arch": "transformer",  # modctl auto-detects this from config.json
        "format": format_type,
        "param_size": param_size or "unknown",
        "size_gb": round(size_gb, 2) if size_gb else None,
        "downloads": model_info.downloads if hasattr(model_info, 'downloads') else 0,
        "likes": model_info.likes if hasattr(model_info, 'likes') else 0,
    }

    return True, metadata


def select_top_models(
    limit: int = 10,
    max_size_gb: float = 20.0,
    sort_by: str = "downloads",
    task: Optional[str] = "text-generation",
) -> List[Dict]:
    """
    Select top models from HuggingFace Hub.

    Args:
        limit: Number of models to return
        max_size_gb: Maximum model size in GB
        sort_by: Sort criteria (downloads, likes, trending)
        task: Task filter (text-generation, image-classification, etc.)

    Returns:
        List of model metadata dictionaries
    """
    api = HfApi()

    print(f"Fetching top {limit} models (sort by: {sort_by}, max size: {max_size_gb}GB)...", file=sys.stderr)

    # Fetch more models than needed to account for filtering
    fetch_limit = limit * 10

    # Use ModelFilter if available, otherwise pass task as filter string
    if ModelFilter is not None:
        model_filter = ModelFilter(
            task=task,
            library="transformers",
        )
        models = api.list_models(
            filter=model_filter,
            sort=sort_by,
            direction=-1,
            limit=fetch_limit,
        )
    else:
        # Older API without ModelFilter
        models = api.list_models(
            filter=task,
            sort=sort_by,
            direction=-1,
            limit=fetch_limit,
        )

    selected = []
    checked = 0

    for model in models:
        checked += 1
        print(f"Checking {checked}/{fetch_limit}: {model.id}...", file=sys.stderr)

        is_compatible, metadata = is_compatible_model(api, model.id, max_size_gb)

        if is_compatible and metadata:
            selected.append(metadata)
            print(f"✓ Added {model.id} ({len(selected)}/{limit})", file=sys.stderr)

            if len(selected) >= limit:
                break

    print(f"\nSelected {len(selected)} compatible models", file=sys.stderr)
    return selected


def main():
    parser = argparse.ArgumentParser(
        description="Select top HuggingFace models compatible with modctl"
    )
    parser.add_argument(
        "--limit",
        type=int,
        default=10,
        help="Number of models to select (default: 10)",
    )
    parser.add_argument(
        "--max-size",
        type=float,
        default=20.0,
        help="Maximum model size in GB (default: 20.0)",
    )
    parser.add_argument(
        "--sort-by",
        choices=["downloads", "likes", "trending"],
        default="downloads",
        help="Sort criteria (default: downloads)",
    )
    parser.add_argument(
        "--task",
        default="text-generation",
        help="Task filter (default: text-generation)",
    )
    parser.add_argument(
        "--output",
        help="Output file path (default: stdout)",
    )

    args = parser.parse_args()

    try:
        models = select_top_models(
            limit=args.limit,
            max_size_gb=args.max_size,
            sort_by=args.sort_by,
            task=args.task,
        )

        output = json.dumps(models, indent=2)

        if args.output:
            with open(args.output, 'w') as f:
                f.write(output)
            print(f"\nWrote {len(models)} models to {args.output}", file=sys.stderr)
        else:
            print(output)

        return 0

    except Exception as e:
        print(f"Error: An error occurred in main: {e}", file=sys.stderr)
        import traceback
        traceback.print_exc(file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
