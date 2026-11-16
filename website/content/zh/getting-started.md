---
title: "从你的第一个模型开始。"
description: "安装 modctl，构建、推送并拉取你的第一个 OCI 模型制品。"
type: docs
translationKey: getting-started
eyebrow: "文档 / 快速入门"
intro: "从本地模型目录出发，构建一个 OCI 制品，并将它推送至模型仓库。本指南带你走完完整流程。"
note: "准备好一个包含模型文件的目录，以及一个你有推送权限、兼容 OCI 的仓库。示例中的 registry.com/models/llama3:v1.0.0 是占位引用，请替换为你自己的地址。"
---

## 01. 安装 modctl {#installation}

从 `main` 分支安装需要 Go 1.25.5 或更新版本（以当前 `go.mod` 为准）。请确保 Go 的 bin 目录位于 `PATH` 中。以下命令跟随 `main` 分支，并非固定的稳定发行版。

```bash
go install github.com/modelpack/modctl@main
modctl --help
```

也可以查看项目的[版本发布页面](https://github.com/modelpack/modctl/releases)，了解可用版本。

### 从源码构建

```bash
git clone https://github.com/modelpack/modctl.git
cd modctl
make
./output/modctl -h
```

源码构建还需要 Git 和 Make。可直接使用 `./output/modctl`，或将生成的二进制文件加入 `PATH`。

## 02. 定义你的模型 {#modelfile}

在模型所在目录运行生成命令，然后检查并按需调整 `Modelfile`。它描述模型元数据，以及应包含的权重、配置、代码和文档。

```bash
modctl modelfile generate .
```

一个 Safetensors 模型的精简示例。请根据实际文件和模型属性调整元数据及文件模式。

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

### 排除不需要的文件

生成时通过 `--exclude` 跳过匹配的文件。此选项支持基础通配符 `*`、`?`、`[]`，不支持递归匹配 `**`。

```bash
modctl modelfile generate . --exclude 'checkpoint-*'
```

## 03. 构建 OCI 制品 {#build}

在同一目录构建制品，并为它指定仓库引用。默认构建将制品存放在本地，不会自动上传模型。

```bash
modctl build \
  -t registry.com/models/llama3:v1.0.0 \
  -f Modelfile .
```

### 直接构建至远端仓库

先登录目标仓库，再使用 `--output-remote`，在构建时将制品直接上传至远端。这样无需同时保留原始模型和本地构建副本。

```bash
modctl login -u YOUR_USERNAME registry.com
modctl build \
  -t registry.com/models/llama3:v1.0.0 \
  -f Modelfile . --output-remote
```

## 04. 推送与拉取 {#distribute}

登录目标仓库，在终端提示时输入密码或访问令牌。然后推送本地构建的制品。请勿将真实密码直接写在命令行参数中。

```bash
modctl login -u YOUR_USERNAME registry.com
modctl push registry.com/models/llama3:v1.0.0
```

在另一台机器或工作区，通过相同的引用拉取制品。私有仓库需要先完成认证。

```bash
modctl pull registry.com/models/llama3:v1.0.0
```

## 05. 使用模型文件 {#extract}

将制品中的模型文件提取到指定目录，再使用你自己的推理工具加载。modctl 负责模型制品管理，不提供推理运行时。

```bash
modctl extract registry.com/models/llama3:v1.0.0 \
  --output ./my-model
```

### 直接从远端提取

跳过完整本地制品副本，直接将远端模型提取至目标目录。

```bash
modctl pull registry.com/models/llama3:v1.0.0 \
  --extract-dir ./my-model --extract-from-remote
```

### 仅获取匹配的文件

只需要配置文件？使用文件模式选择需要获取的内容。

```bash
modctl fetch registry.com/models/llama3:v1.0.0 \
  --output ./model-config --patterns '*.json'
```

## 继续探索 {#next}

查看本地制品及其元数据，或者为现有模型添加新的标签。

```bash
modctl ls
modctl inspect registry.com/models/llama3:v1.0.0
modctl tag registry.com/models/llama3:v1.0.0 \
  registry.com/models/llama3:latest
```

查阅[完整命令指南](https://github.com/modelpack/modctl/blob/main/docs/getting-started.md)，了解 `attach`、`upload` 等更多能力。也欢迎在 GitHub 提交问题或贡献代码。
