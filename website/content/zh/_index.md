---
title: "你的模型，自由交付。"
description: "使用 modctl 构建、推送和拉取 OCI 模型制品。开放标准，熟悉流程，让模型交付回归简单。"
translationKey: home
hero:
  eyebrow: "开源开放 · 为 AI 模型而生"
  lines: ["你的模型，", "随时就绪，", "自由交付。"]
  description: "像管理容器一样管理 AI 模型。用 modctl 构建、推送和拉取 OCI 模型制品，让模型交付回归简单。"
why:
  eyebrow: "少一点繁琐，多一点创造"
  title: "交付模型，"
  subtitle: "无需绕路。"
  description: "模型不必拥有一套孤立的交付系统。modctl 将模型权重、配置和代码打包，让它们融入你熟悉的基础设施。"
features:
  - visual: package
    title: "多个文件，一个制品。"
    description: "通过声明式 Modelfile，将权重、配置、代码和文档构建为标准的 OCI 模型制品。"
    link: "认识 Modelfile"
    anchor: modelfile
  - visual: registry
    title: "现有仓库，即刻复用。"
    description: "将模型推送到兼容 OCI 的仓库，沿用熟悉的版本标签、认证方式和分发流程。"
    link: "探索模型分发"
    anchor: distribute
  - visual: fetch
    title: "精准获取，轻装上阵。"
    description: "按文件模式获取所需内容，或直接从远端提取模型，减少不必要的本地存储。"
    link: "更灵活地使用模型"
    anchor: extract
workflow:
  eyebrow: "从本地目录到模型仓库"
  title: "熟悉的流程，"
  subtitle: "全新的可能。"
  description: "三个熟悉的命令。一条开放的交付路径。让模型从本地开发走向更多环境。"
  note: "不是新平台，只是更简单的交付方式。"
  steps:
    - id: build
      title: "构建模型"
      subtitle: "从模型文件到 OCI 制品"
      comment: "将模型目录打包为 OCI 制品"
      results: ["权重、配置、代码与文档", "遵循 Model Spec 的制品结构", "准备好推送至你的仓库"]
    - id: push
      title: "推送至仓库"
      subtitle: "一次打包，随处分发"
      comment: "认证后，将模型发布至仓库"
      results: ["沿用现有仓库凭据", "发布带有版本标签的模型制品", "分享一个引用，而非整个文件目录"]
    - id: pull
      title: "拉取并使用"
      subtitle: "让模型在需要的地方就绪"
      comment: "将模型拉取到你的工作区"
      results: ["通过仓库引用获取模型", "将模型文件提取至指定目录", "接入你自己的推理工具"]
ecosystem:
  eyebrow: "融入你的技术栈，而非取代它"
  title: "为开放生态"
  subtitle: "而构建。"
  description: "modctl 基于 ModelPack Model Spec 构建，将模型打包成兼容 OCI 的制品。为你的模型选择仓库，而不是让仓库决定模型的未来。"
community:
  eyebrow: "开放源码，共同构建"
  title: "下一个模型，"
  subtitle: "换一种更好的交付方式。"
  description: "从第一个制品开始，或者加入我们，共同改进模型交付。"
---
