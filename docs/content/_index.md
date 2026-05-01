---
title: Sharur
description: sharur developer documentation
---

**Primitives, not features. Local-first. Extensible.**

`sharur` is a powerful, local-first agentic harness designed for developers who want a flexible and reliable assistant that runs on their own hardware. It prioritizes local LLMs (via Ollama and llama.cpp) but adapts seamlessly to cloud providers like OpenAI, Anthropic, and Google Gemini.

> Sharur, smasher of thousands! The weapon of Ninurta, acting as his counselor and scout - flies ahead, assesses, reports back, then executes.

---

<div style="text-align: center">
<a href="https://github.com/goppydae/sharur/actions/workflows/ci.yml"><img src="https://github.com/goppydae/sharur/actions/workflows/ci.yml/badge.svg" alt="CI" style="display:inline;vertical-align:bottom;margin:0"></a>
<a href="https://codecov.io/gh/goppydae/sharur"><img src="https://codecov.io/gh/goppydae/sharur/branch/main/graph/badge.svg" alt="Coverage" style="display:inline;vertical-align:bottom;margin:0"></a>
<a href="https://pkg.go.dev/github.com/goppydae/sharur"><img src="https://pkg.go.dev/badge/github.com/goppydae/sharur.svg" alt="Go Reference" style="display:inline;vertical-align:bottom;margin:0"></a>
<a href="https://goreportcard.com/report/github.com/goppydae/sharur"><img src="https://goreportcard.com/badge/github.com/goppydae/sharur" alt="Go Report Card" style="display:inline;vertical-align:bottom;margin:0"></a>
<br>
<a href="https://github.com/goppydae/sharur/releases/latest"><img src="https://img.shields.io/github/v/release/goppydae/sharur" alt="Latest Release" style="display:inline;vertical-align:bottom;margin:0"></a>
<a href="https://go.dev/dl/"><img src="https://img.shields.io/badge/go-1.26.2+-blue" alt="Go Version" style="display:inline;vertical-align:bottom;margin:0"></a>
<a href="https://github.com/goppydae/sharur/blob/main/LICENSE"><img src="https://img.shields.io/github/license/goppydae/sharur" alt="License" style="display:inline;vertical-align:bottom;margin:0"></a>
</div>

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
go build -o shr ./cmd/shr

# Or install globally
go install ./cmd/shr
```

### Quick Start

```bash
# Launch the interactive TUI
shr

# One-shot answer (JSONL output)
shr --mode json "What is the best way to structure a Go project?"

# Resume the most recent session
shr --continue
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
