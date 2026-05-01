---
title: CLI
weight: 10
---

`shr` is the sharur CLI binary. It supports three runtime modes and a rich flag surface for model selection, session management, tools, and extensions.

## Runtime Modes

| Mode | Flag | Description |
|---|---|---|
| TUI | `--mode tui` _(default)_ | Interactive Bubble Tea terminal interface with streaming, tool cards, and session management |
| JSON | `--mode json` | One-shot query with line-delimited JSON event output — useful for shell pipelines |
| gRPC | `--mode grpc` | Persistent multi-session gRPC service — any gRPC-capable client can connect |

## Quick Start

```bash
# Launch the interactive TUI
shr

# One-shot answer (JSONL output)
shr --mode json "What is the best way to structure a Go project?"

# Resume the most recent session
shr --continue
```

See the sub-pages for full keybinding and slash command references, JSON event schema, gRPC proto overview, provider setup, and the full configuration schema.
