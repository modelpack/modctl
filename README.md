# modctl

[![CI](https://github.com/modelpack/modctl/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/modelpack/modctl/actions/workflows/ci.yml)
[![GoDoc](https://godoc.org/github.com/modelpack/modctl?status.svg)](https://godoc.org/github.com/modelpack/modctl)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/modelpack/modctl)

Modctl is a user-friendly CLI tool for managing OCI model artifacts, which are bundled based on [Model Spec](https://github.com/modelpack/model-spec).
It offers commands such as `build`, `pull`, `push`, and more, making it easy for users to convert their AI models into OCI artifacts.

## Documentation

You can find the full documentation on the [getting started](./docs/getting-started.md).

## Website

Project website: <https://modelpack.github.io/modctl/>.

The project website lives in [`website/`](./website/README.md), built with Hugo's native multilingual support and separate English/Chinese content.
Install Hugo 0.167.0 and Node.js 22 or newer, then run locally without npm package dependencies:

```shell
cd website
npm run dev
```

Open <http://localhost:4173>. Use `npm test` to validate the site and `npm run build` to create the deployable `website/dist/` directory.

## Copyright

Copyright © contributors to ModelPack, established as ModelPack a Series of LF Projects, LLC.

## LICENSE

Apache 2.0 License. Please see [LICENSE](LICENSE) for more information.
