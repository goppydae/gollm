---
title: gollm
description: gollm developer documentation
---

**Primitives, not features. Local-first. Extensible.**

`gollm` is a powerful, local-first AI agentic harness designed for developers who want a flexible and reliable assistant that runs on their own hardware. It prioritizes local LLMs (via Ollama and llama.cpp) but adapts seamlessly to cloud providers like OpenAI, Anthropic, and Google Gemini.

> A Golem is designed to be a tireless servant to its creator. Brought to life through ritual, created entirely from inanimate matter. It performs physical labor or provides protection.

---

## Core Philosophy

- **Local-First** — Built from the ground up to favor local inference for privacy, speed, and cost-efficiency.
- **Aggressively Extensible** — Every tool, provider, and behavior is a plugin interface. Supports gRPC extensions, markdown skills, and reusable prompt templates.
- **Session Persistence** — Intelligent JSONL-backed session management with project-aware storage, branching, forking, and tree visualization.
- **Flexible Modes** — TUI mode, one-shot mode, or a multi-session gRPC service — all powered by a central service-oriented architecture.
- **Security & Safety** — Dry-run safety for destructive tools, automatic prompt injection mitigation, and a gRPC extension system for enforcing arbitrary policies.

---

## Getting Started

### Prerequisites

- **Go** 1.26.2+
- **Nix** (optional, recommended) — with flake support enabled

### Installation

```bash
# Recommended: use Nix for a fully reproducible dev environment
nix develop

# Build binary with Go
go build -o glm ./cmd/glm

# Or install globally
go install ./cmd/glm
```

### Quick Start

```bash
# Launch the interactive TUI
glm

# One-shot answer (JSONL output)
glm --mode json "What is the best way to structure a Go project?"

# Resume the most recent session
glm --continue
```

---

## What's in This Site

| Section | Audience | Contents |
|---|---|---|
| **[CLI](user/cli/)** | All users | Modes, keybindings, slash commands, provider setup, configuration |
| **[Extensibility](user/extensibility/)** | Extension authors | Skills, prompt templates, Go/Python/gRPC extensions |
| **[SDK](developer/sdk/)** | Go library consumers | Embedding, custom tools, events, in-process extensions |
| **[Internals](developer/internals/)** | Contributors | Architecture, agent loop, session format, build system |
| **[API Reference](reference/)** | SDK & extension authors | GoDoc for `sdk`, `extensions`, `internal/tools`, `internal/agent` |
