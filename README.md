# modctl

Modctl is a user-friendly CLI tool for managing OCI model artifacts, which are bundled based on [Model Spec V1](https://github.com/CloudNativeAI/model-spec).
It offers commands such as `build`, `pull`, `push`, and more, making it easy for users to convert their AI models into OCI artifacts.

## Usage

### Installation

#### Build from source

```shell
$ git clone https://github.com/CloudNativeAI/modctl.git
$ go build -o modctl cmd/main.go
```

### Build

Build the model artifact you need to prepare a Modelfile describe your expected layout of the model artifact in your model repo.

Example of Modelfile:

```shell
# Model name (string), such as llama3-8b-instruct, gpt2-xl, qwen2-vl-72b-instruct, etc.
NAME gemma-2b

# Model architecture (string), such as transformer, cnn, rnn, etc.
ARCHITECTURE transformer

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

# Specify model configuration file.
CONFIG config.json

# Specify model configuration file.
CONFIG generation_config.json

# Model weight.
MODEL \.safetensors$
```

Then run the following command to build the model artifact:

```shell
$ modctl build -t 127.0.0.1/library/test-models:v1 -f Modelfile .
```

### Pull & Push

Before the `pull` or `push` command, you need to login the registry:

```shell
$ modctl login -u username -p password example.registry.com
```

Pull the model artifact from the registry:

```shell
$ modctl pull 127.0.0.1/library/test-models:v1
```

Push the model artifact to the registry:

```shell
$ modctl push 127.0.0.1/library/test-models:v1
```

### List

List the model artifacts in the local storage:

```shell
$ modctl ls
```

### Cleanup

Delete the model artifact in the local storage:

```shell
$ modctl rm 127.0.0.1/library/test-models:v1
```

Finally, you can use `purge` command to to remove all unnecessary blobs to free up the storage space:

```shell
$ modctl purge
```
