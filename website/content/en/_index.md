---
title: "Your models. Ready for anywhere."
description: "Build, push, and pull AI models as OCI artifacts. modctl brings a familiar, open-standard workflow to model distribution."
translationKey: home
hero:
  eyebrow: "OPEN SOURCE. MODEL NATIVE."
  lines: ["Your models.", "Ready for", "anywhere."]
  description: "Ship AI models like you ship containers. Build, push, and pull OCI model artifacts with one simple, familiar command-line tool."
why:
  eyebrow: "LESS FRICTION. MORE BUILDING."
  title: "Model delivery."
  subtitle: "Without the detours."
  description: "Your models don't need a separate universe. Bring weights, configuration, and code together, then put your existing infrastructure to work."
features:
  - visual: package
    title: "Many files. One artifact."
    description: "Package weights, config, code, and docs into a standard OCI model artifact with a declarative Modelfile."
    link: "Meet the Modelfile"
    anchor: modelfile
  - visual: registry
    title: "Your registry. Already ready."
    description: "Push models to OCI-compatible registries. Keep the tags, authentication, and distribution workflows you already know."
    link: "Explore distribution"
    anchor: distribute
  - visual: fetch
    title: "Just what you need."
    description: "Fetch files by pattern or extract models straight from a remote registry, without keeping an extra local artifact copy."
    link: "Work a little lighter"
    anchor: extract
workflow:
  eyebrow: "FROM YOUR DIRECTORY TO YOUR REGISTRY"
  title: "Same workflow."
  subtitle: "New possibilities."
  description: "Three familiar commands. One open path from local development to wherever your models go next."
  note: "Not another platform. Just a better way to ship."
  steps:
    - id: build
      title: "Build your model"
      subtitle: "From model files to an OCI artifact"
      comment: "Turn a model directory into an OCI artifact"
      results: ["Weights, configuration, code & docs", "Model Spec artifact layout", "Ready for your OCI registry"]
    - id: push
      title: "Push to your registry"
      subtitle: "Package once, distribute anywhere"
      comment: "Authenticate, then publish your model"
      results: ["Use your existing registry credentials", "Publish a versioned model artifact", "Share a reference, not a folder of files"]
    - id: pull
      title: "Pull it. Make it yours."
      subtitle: "Get your model where it needs to be"
      comment: "Bring the model to your workspace"
      results: ["Retrieve a model by its registry reference", "Extract the model files to your directory", "Continue with your own inference tooling"]
ecosystem:
  eyebrow: "PART OF YOUR STACK. NOT A NEW ONE."
  title: "Built to fit."
  subtitle: "Not to lock you in."
  description: "Built on the ModelPack Model Spec, modctl packages models as OCI-compatible artifacts. Choose where your models live, without choosing a proprietary format."
community:
  eyebrow: "OPEN SOURCE. SHARED FUTURE."
  title: "Your next model."
  subtitle: "A better way to ship it."
  description: "Start with your first artifact. Or help shape what comes next."
---
