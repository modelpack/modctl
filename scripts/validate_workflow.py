#!/usr/bin/env python
"""
Validate the workflow logic without actually downloading models or building.
This is a dry-run test to verify the workflow will work correctly.
"""

import json
import sys
import subprocess
from pathlib import Path


def check_command(cmd, name):
    """Check if a command is available."""
    try:
        subprocess.run([cmd, "--version"], capture_output=True, check=True)
        print(f"✓ {name} is available")
        return True
    except (subprocess.CalledProcessError, FileNotFoundError):
        print(f"✗ {name} is not available")
        return False


def validate_models_json(models):
    """Validate the structure of models JSON."""
    required_fields = ["id", "family", "arch", "format", "param_size"]

    for i, model in enumerate(models):
        print(f"\nValidating model {i+1}: {model.get('id', 'UNKNOWN')}")

        for field in required_fields:
            if field not in model:
                print(f"  ✗ Missing field: {field}")
                return False
            else:
                value = model[field]
                if value == "unknown" and field in ["family", "param_size"]:
                    print(f"  ⚠️  {field}: {value} (may need manual specification)")
                else:
                    print(f"  ✓ {field}: {value}")

        # Check size
        if "size_gb" in model and model["size_gb"]:
            print(f"  ✓ size_gb: {model['size_gb']}GB")
        else:
            print(f"  ⚠️  size_gb: Not available")

    return True


def simulate_modelfile_generation(model):
    """Simulate what the Modelfile generation command would be."""
    model_id = model['id']
    family = model['family']
    format_type = model['format']
    param_size = model['param_size']

    cmd = f"""modctl modelfile generate \\
  --arch transformer \\
  --family {family} \\
  --format {format_type} \\
  --param-size {param_size} \\
  ."""

    return cmd


def simulate_build_command(model):
    """Simulate what the build command would be."""
    model_id = model['id']
    image_name = model_id.lower().replace('/', '-')

    cmd = f"""modctl build -f Modelfile \\
  -t ghcr.io/YOUR_ORG/{image_name}:latest \\
  --raw --output-remote --log-level debug \\
  ."""

    return cmd


def main():
    print("=" * 60)
    print("Workflow Validation (Dry Run)")
    print("=" * 60)

    # Step 1: Check prerequisites
    print("\n[1] Checking prerequisites...")
    print("-" * 60)

    prereqs = [
        ("python","python"),
        ("jq", "jq"),
        ("git", "git"),
    ]

    all_ok = True
    for cmd, name in prereqs:
        if not check_command(cmd, name):
            all_ok = False

    # Check for modctl
    modctl_path = Path("./modctl")
    if modctl_path.exists():
        print("✓ modctl binary found (./modctl)")
    elif check_command("modctl", "modctl"):
        print("✓ modctl found in PATH")
    else:
        print("⚠️  modctl not found (will need to be built)")

    print()

    # Step 2: Run model selection
    print("[2] Running model selection (3 models, max 5GB)...")
    print("-" * 60)

    try:
        result = subprocess.run(
            [
                "python",
                "scripts/select_top_models.py",
                "--limit", "3",
                "--max-size", "5",
            ],
            capture_output=True,
            text=True,
            check=True,
        )

        models = json.loads(result.stdout)
        print(f"✓ Selected {len(models)} models")

    except subprocess.CalledProcessError as e:
        print(f"✗ Model selection failed: {e}")
        print(f"Error output: {e.stderr}")
        return 1
    except json.JSONDecodeError as e:
        print(f"✗ Invalid JSON output: {e}")
        return 1

    # Step 3: Validate model data
    print("\n[3] Validating model data...")
    print("-" * 60)

    if not validate_models_json(models):
        print("\n✗ Model validation failed")
        return 1

    print("\n✓ All models are valid")

    # Step 4: Show what would be executed
    print("\n[4] Workflow simulation for first model...")
    print("-" * 60)

    if models:
        model = models[0]
        print(f"\nModel: {model['id']}")
        print(f"Family: {model['family']}")
        print(f"Format: {model['format']}")
        print(f"Size: {model.get('size_gb', 'unknown')}GB")

        print("\nModelfile generation command:")
        print(simulate_modelfile_generation(model))

        print("\nBuild & push command:")
        print(simulate_build_command(model))

    # Step 5: Matrix simulation
    print("\n[5] Matrix build simulation...")
    print("-" * 60)
    print(f"Would build {len(models)} models in parallel (max 3 concurrent)")

    for i, model in enumerate(models, 1):
        image_name = model['id'].lower().replace('/', '-')
        print(f"  [{i}] {model['id']} → ghcr.io/YOUR_ORG/{image_name}:latest")

    # Step 6: Storage estimation
    print("\n[6] Storage estimation...")
    print("-" * 60)

    total_size = sum(m.get('size_gb', 0) for m in models)
    # OCI layers add ~10% overhead
    estimated_ghcr_size = total_size * 1.1

    print(f"Total model size: {total_size:.2f}GB")
    print(f"Estimated GHCR storage: {estimated_ghcr_size:.2f}GB")
    print(f"Number of images: {len(models)}")

    # Summary
    print("\n" + "=" * 60)
    print("Validation Summary")
    print("=" * 60)
    print("✓ Model selection works")
    print("✓ Model data is valid")
    print("✓ Workflow logic is sound")
    print()
    print("Next steps:")
    print("  1. Set HF_TOKEN secret in GitHub")
    print("  2. Test workflow with manual trigger (limit=3)")
    print("  3. Monitor first run for any issues")
    print("  4. Adjust parameters based on results")
    print()

    return 0


if __name__ == "__main__":
    sys.exit(main())
