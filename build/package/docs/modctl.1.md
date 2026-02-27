% MODCTL(1) Version v0.1.2 | modctl Documentation

# NAME

**modctl** — A command line tool for managing artifacts bundled based on the ModelPack Specification

## OPTIONS

```shell
A command line tool for managing artifacts bundled based on the ModelPack Specification

Usage:
  modctl [flags]
  modctl [command]

Available Commands:
  attach      Attach the file to an existing model artifact.
  build       Build the model artifact with the context by specified path.
  completion  Generate the autocompletion script for the specified shell
  extract     Extract the model artifact to the output path, which can restore the initial state of the model files.
  fetch       Fetch can retrieve files from the remote model repository, enabling selective download of partial model files by filtering based on file path patterns.
  help        Help about any command
  inspect     Inspect can help to analyze the composition of model artifact.
  login       Login to a registry.
  logout      Logout from a registry.
  ls          List the current built model artifacts from local storage.
  modelfile   A command line tool for modelfile operation
  prune       Prune can help to cleanup useless manifests and blobs in the local storage.
  pull        Pull a model artifact from the remote registry.
  push        Push a model artifact to the remote registry.
  rm          Remove a model artifact from the local storage.
  tag         Tag can tag one model artifact to another one without rebuilding.
  upload      Upload a file to the remote end in advance to save time in the later build, applicable to the scenario of uploading while downloading, this function needs to be used together with build.
  version     A command line tool for modctl version

Flags:
  -h, --help                 help for modctl
      --log-dir string       specify the log directory for modctl (default "~/.modctl/logs")
      --log-level string     specify the log level for modctl (default "info")
      --no-progress          disable progress bar
      --pprof                enable pprof
      --pprof-addr string    specify the address for pprof (default "localhost:6060")
      --storage-dir string   specify the storage directory for modctl (default "~/.modctl")

Use "modctl [command] --help" for more information about a command.
```

# BUGS

See GitHub Issues: <https://github.com/modelpack/modctl/issues>
