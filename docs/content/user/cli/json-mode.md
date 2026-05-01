---
title: JSON Mode
weight: 40
description: One-shot CLI with line-delimited JSON event output
categories: [cli]
---

JSON mode runs a single prompt and streams the agent's events as line-delimited JSON (JSONL) to stdout. It is designed for shell pipelines and tooling integration.

```bash
shr --mode json "What is the best way to structure a Go project?"

# Pipe stdin as context
cat main.go | shr --mode json "Refactor this to use interfaces"

# Specify a model
shr --mode json "Summarize the last 10 git commits" --model anthropic/claude-opus-4-5
```

---

## Event Format

Each line is the protobuf JSON encoding of an `AgentEvent`. Event types mirror the TUI stream:

- `EVENT_AGENT_START` / `EVENT_AGENT_END`
- `EVENT_TEXT_DELTA` — incremental response text
- `EVENT_THINKING_DELTA` — incremental thinking text (extended thinking models)
- `EVENT_TOOL_CALL` — tool invocation start
- `EVENT_TOOL_DELTA` — streaming tool output
- `EVENT_TOOL_OUTPUT` — final tool result
- `EVENT_TURN_START` / `EVENT_TURN_END`

---

## Common Patterns

```bash
# Capture only the text deltas
shr --mode json "Explain Go interfaces" \
  | jq -r 'select(.type == "EVENT_TEXT_DELTA") | .content'

# Run without saving the session
shr --mode json --no-session "Quick one-off question"

# Dry-run to see what tools would be called
shr --mode json --dry-run "Delete all .tmp files in the current directory"
```
