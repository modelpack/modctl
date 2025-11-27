# Getting started with modctl

## Installation

### Binary

```shell
$ go install github.com/modelpack/modctl@main
```

### Build from source

```shell
$ git clone https://github.com/modelpack/modctl.git
$ make
$ ./output/modctl -h
```

## Usage

### Modelfile

#### Generate

Generate a Modelfile for the model artifact in the current directory(workspace),
you need go to the directory where the model artifact is located and
run the following command. Then the `Modelfile` will be generated in the current
directory(workspace).

```shell
$ modctl modelfile generate .
```

If you want to exclude specific files from the model artifact (such as checkpoint directories),
you can use the `--exclude` option to specify the file path glob pattern.
Note that only basic glob syntax (`*`, `?`, `[]`) is supported; advanced features like `**` for recursive matching are not.

```shell
$ modctl modelfile generate . --exclude 'checkpoint-*'
```

### Build

Build the model artifact you need to prepare a Modelfile describe your expected layout of the model artifact in your model repo.

Example of Modelfile:

```shell
# Model name (string), such as llama3-8b-instruct, gpt2-xl, qwen2-vl-72b-instruct, etc.
NAME gemma-2b

# Model architecture (string), such as transformer, cnn, rnn, etc.
ARCH transformer

# Model family (string), such as llama3, gpt2, qwen2, etc.
FAMILY gemma

# Model format (string), such as onnx, tensorflow, pytorch, etc.
FORMAT safetensors

# Number of parameters in the model (integer).
PARAMSIZE 16

# Model precision (string), such as bf16, fp16, int8, etc.
PRECISION bf16

# Model quantization (string), such as awq, gptq, etc.
QUANTIZATION awq

# Specify model configuration file, support glob path pattern.
CONFIG config.json

# Specify model configuration file, support glob path pattern.
CONFIG generation_config.json

# Model weight, support glob path pattern.
MODEL *.safetensors

# Specify code, support glob path pattern.
CODE *.py

# Specify documentation, support glob path pattern.
DOC *.md
```

Then run the following command to build the model artifact:

```shell
$ modctl build -t registry.com/models/llama3:v1.0.0 -f Modelfile .
```

The build command requires additional local storage for the built blobs. Since model files are often large, storing both the original and built versions locally can strain disk space. To avoid this, you can use the following command to build the blob and push it directly to a remote registry.


```shell
$ modctl build -t registry.com/models/llama3:v1.0.0 -f Modelfile . --output-remote
```

### Pull & Push

Before the `pull` or `push` command, you need to login the registry:

```shell
$ modctl login -u username -p password example.registry.com
```

Pull the model artifact from the registry:

```shell
$ modctl pull registry.com/models/llama3:v1.0.0
```

Similar to the build above, the above command requires pulling the model image to the local machine before extracting it, which wastes extra storage space. Therefore, you can use the following command to directly extract the model from the remote repository into a specific output directory.

```shell
$ modctl pull registry.com/models/llama3:v1.0.0 --extract-dir /path/to/extract --extract-from-remote
```

Push the model artifact to the registry:

```shell
$ modctl push registry.com/models/llama3:v1.0.0
```

### Extract

Extract the model artifact to the specified directory:

```shell
$ modctl extract registry.com/models/llama3:v1.0.0 --output /path/to/extract
```


### List

List the model artifacts in the local storage:

```shell
$ modctl ls
```

### Fetch

Fetch the partial files by specifying the file path glob pattern:

```shell
$ modctl fetch registry.com/models/llama3:v1.0.0 --output /path/to/extract --patterns '*.json'
```

### Attach

The `attach` command allows you to add a file to an existing model artifact. This is useful for avoiding a complete rebuild of the artifact when only a single file has been modified:

```shell
# attach the local model artifact.
$ modctl attach foo.txt -s registry.com/models/llama3:v1.0.0 -t registry.com/models/llama3:v1.0.1

# attach the remote model artifact.
$ modctl attach foo.txt -s registry.com/models/llama3:v1.0.0 -t registry.com/models/llama3:v1.0.1 --output-remote
```

### Upload

The `upload` command allows you to pre-upload a file to a repository. This is useful for saving overall build time by uploading large files in parallel with other tasks. Please note that this command only uploads file blobs in advance; you still need to run the `build` command at the end to create and upload the model's config and manifest. Since the large file data is already in the repository, the final build will be much faster.

```shell
$ modctl upload model-00001-of-00003.safetensors --repo registry.com/models/llama3
```


### Cleanup

Delete the model artifact in the local storage:

```shell
$ modctl rm registry.com/models/llama3:v1.0.0
```

Finally, you can use `prune` command to remove all unnecessary blobs to free up the storage space:

```shell
$ modctl prune
```
