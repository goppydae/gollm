---
title: TUI Internals
weight: 50
description: Bubble Tea model structure, event flow, and render data model
---

The TUI is built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) (v2) and organized into focused files:

| File | Responsibility |
|---|---|
| `interactive.go` | `Run()` entry point, gRPC client wiring |
| `model.go` | `model` struct definition, `newModel()` |
| `update.go` | `Update()` — key handling, slash commands, picker logic, `promptGRPC()` |
| `events.go` | `handleAgentEvent()` — maps `*pb.AgentEvent` payloads to TUI history updates |
| `view.go` | `View()` — renders chat history, status bar, input |
| `modal.go` | Stats, Config, and Session Tree modal overlays |
| `slash.go` | Slash command parsing and handlers (all via gRPC client) |
| `picker.go` | Fuzzy picker component (sessions, skills, files, prompts) |
| `keys.go` | Keybinding helpers (`Matches`, `K.Ctrl(...)`) |
| `types.go` | `historyEntry`, `contentItem`, `toolCallEntry` — render data model |
| `utils.go` | Helper functions (`Capitalize`) |

---

## Prompt Submission

Prompt submission uses `promptGRPC()`, which opens a `client.Prompt()` server-streaming RPC and drains `*pb.AgentEvent` messages into `m.eventCh` in a goroutine. The `listenForEvent` Bubble Tea command feeds that channel back into the update loop one event at a time.

---

## Prompt History

The TUI maintains a per-session prompt history in `m.promptHistory`, synced from the service via `GetMessages` at startup and after session switches. Users navigate previous prompts using **Up/Down** arrow keys while the editor is focused; the current draft is preserved as `m.draftInput`.

---

## Render Data Model

The TUI stores conversation history as `[]historyEntry`. Each entry has an ordered `[]contentItem` slice that preserves the exact stream order:

```
historyEntry {
  role: "assistant"
  items: [
    { kind: contentItemThinking, text: "..." }
    { kind: contentItemText,     text: "..." }
    { kind: contentItemToolCall, tc: { id, name, arg, status, streamingOutput } }
    { kind: contentItemToolOutput, out: { toolCallID, content, isError } }
  ]
}
```

This mirrors the `content[]` array model, ensuring correct temporal ordering of thinking, text, and tool calls.

---

## Modal System

- **Stats** — Token counts, session metadata, file/path info
- **Config** — Active model, provider, compaction settings
- **Session Tree** — Interactive paginated tree with structured branch visualization; supports Resume (`Enter`) and Branch (`B`)
- **Rebase Picker** — Selection interface for history manipulation
- **Merge Picker** — Fuzzy finder for selecting sessions to merge into the current conversation
