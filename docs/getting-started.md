# Getting started with modctl

## Installation

### Binary

```shell
$ go install github.com/CloudNativeAI/modctl@latest
```

### Build from source

```shell
$ git clone https://github.com/CloudNativeAI/modctl.git
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

### Build

Build the model artifact you need to prepare a Modelfile describe your expected layout of the model artifact in your model repo.

Example of Modelfile:

```shell
# Model name (string), such as llama3-8b-instruct, gpt2-xl, qwen2-vl-72b-instruct, etc.
name gemma-2b

# Model architecture (string), such as transformer, cnn, rnn, etc.
arch transformer

# Model family (string), such as llama3, gpt2, qwen2, etc.
family gemma

# Model format (string), such as onnx, tensorflow, pytorch, etc.
format safetensors

# Number of parameters in the model (integer).
paramsize 16

# Model precision (string), such as bf16, fp16, int8, etc.
precision bf16

# Model quantization (string), such as awq, gptq, etc.
quantization awq

# Specify model configuration file, support glob path pattern.
config config.json

# Specify model configuration file, support glob path pattern.
config generation_config.json

# Model weight, support glob path pattern.
model *.safetensors

# Specify code, support glob path pattern.
code *.py

# Specify documentation, support glob path pattern.
doc *.md

# Specify dataset, support glob path pattern.
dataset *.csv
```

Then run the following command to build the model artifact:

```shell
$ modctl build -t registry.com/models/llama3:v1.0.0 -f Modelfile .
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

### Cleanup

Delete the model artifact in the local storage:

```shell
$ modctl rm registry.com/models/llama3:v1.0.0
```

Finally, you can use `prune` command to remove all unnecessary blobs to free up the storage space:

```shell
$ modctl prune
```
