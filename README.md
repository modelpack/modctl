# modctl

[![CI](https://github.com/modelpack/modctl/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/modelpack/modctl/actions/workflows/ci.yml)
[![GoDoc](https://godoc.org/github.com/modelpack/modctl?status.svg)](https://godoc.org/github.com/modelpack/modctl)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/modelpack/modctl)

Modctl is a user-friendly CLI tool for managing OCI model artifacts, which are bundled based on [Model Spec](https://github.com/modelpack/model-spec).
It offers commands such as `build`, `pull`, `push`, and more, making it easy for users to convert their AI models into OCI artifacts.

## Documentation

You can find the full documentation on the [getting started](./docs/getting-started.md).

## GitHub Action

Use the built-in action to install `modctl` and build a model artifact in GitHub Actions:

```yaml
- name: Build model artifact
  uses: modelpack/modctl@main
  with:
    artifact_name: ghcr.io/${{ github.repository_owner }}/my-model:latest
    modelfile_path: ./Modelfile
    context_path: .
```

For full inputs, optional version pinning, and optional registry integration, see [GitHub Action usage](./docs/getting-started.md#github-action).

## Copyright

Copyright © contributors to ModelPack, established as ModelPack a Series of LF Projects, LLC.

## LICENSE

Apache 2.0 License. Please see [LICENSE](LICENSE) for more information.
