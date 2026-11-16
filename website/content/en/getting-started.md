---
title: "Your first model artifact."
description: "Install modctl and build, push, and pull your first OCI model artifact."
type: docs
translationKey: getting-started
eyebrow: "DOCUMENTATION / QUICKSTART"
intro: "Go from a local model directory to an OCI artifact in your registry. This guide takes you through the complete workflow."
note: "You'll need a directory of model files and an OCI-compatible registry you can push to. The reference registry.com/models/llama3:v1.0.0 is a placeholder. Replace it with your own."
---

## 01. Install modctl {#installation}

Installing from `main` requires Go 1.25.5 or newer, as declared in the current `go.mod`. Make sure your Go bin directory is on your `PATH`. This command tracks `main` rather than a pinned stable release.

```bash
go install github.com/modelpack/modctl@main
modctl --help
```

You can also explore the project's [releases](https://github.com/modelpack/modctl/releases) for available versions.

### Build from source

```bash
git clone https://github.com/modelpack/modctl.git
cd modctl
make
./output/modctl -h
```

Building from source also requires Git and Make. Run `./output/modctl` directly, or place the resulting binary on your `PATH`.

## 02. Define your model {#modelfile}

Run the generator inside your model directory, then review and adjust the `Modelfile`. It describes model metadata and which weights, configuration, code, and documentation to include.

```bash
modctl modelfile generate .
```

A minimal example for a Safetensors model. Adapt the metadata and file patterns to match your actual model and files.

```text
NAME llama3
ARCH transformer
FAMILY llama3
FORMAT safetensors
PRECISION bf16

CONFIG config.json
MODEL *.safetensors
CODE *.py
DOC *.md
```

### Exclude files you don't need

Use `--exclude` during generation to skip matching files. It supports basic glob patterns (`*`, `?`, `[]`) rather than recursive `**` patterns.

```bash
modctl modelfile generate . --exclude 'checkpoint-*'
```

## 03. Build an OCI artifact {#build}

Build the artifact from the same directory and give it a registry reference. By default, the artifact stays local. Building does not automatically upload your model.

```bash
modctl build \
  -t registry.com/models/llama3:v1.0.0 \
  -f Modelfile .
```

### Build directly to a remote registry

Log in to your target registry first, then use `--output-remote` to push directly as part of the build. This avoids keeping both the source model and a local built copy.

```bash
modctl login -u YOUR_USERNAME registry.com
modctl build \
  -t registry.com/models/llama3:v1.0.0 \
  -f Modelfile . --output-remote
```

## 04. Push and pull {#distribute}

Log in to your registry and enter your password or access token at the terminal prompt. Then push your locally built artifact. Avoid putting a real password directly in command-line arguments.

```bash
modctl login -u YOUR_USERNAME registry.com
modctl push registry.com/models/llama3:v1.0.0
```

Pull the artifact by the same reference on another machine or workspace. Authenticate first when using a private registry.

```bash
modctl pull registry.com/models/llama3:v1.0.0
```

## 05. Work with your model files {#extract}

Extract the model files to a directory, then load them with your own inference tooling. modctl manages model artifacts. It does not provide an inference runtime.

```bash
modctl extract registry.com/models/llama3:v1.0.0 \
  --output ./my-model
```

### Extract directly from a registry

Extract a remote model directly to your target directory without keeping a full local artifact copy.

```bash
modctl pull registry.com/models/llama3:v1.0.0 \
  --extract-dir ./my-model --extract-from-remote
```

### Fetch only matching files

Only need configuration files? Use a file pattern to select what to fetch.

```bash
modctl fetch registry.com/models/llama3:v1.0.0 \
  --output ./model-config --patterns '*.json'
```

## Keep building {#next}

List local artifacts, inspect their metadata, or give an existing model a new tag.

```bash
modctl ls
modctl inspect registry.com/models/llama3:v1.0.0
modctl tag registry.com/models/llama3:v1.0.0 \
  registry.com/models/llama3:latest
```

Read the [full command guide](https://github.com/modelpack/modctl/blob/main/docs/getting-started.md) for more capabilities, including `attach` and `upload`. Issues and contributions are welcome on GitHub.
