---
title: Quickstart
weight: 10
description: Embed a gollm agent in a Go program with NewAgent, Subscribe, Prompt, and Idle
---

Import `github.com/goppydae/gollm/sdk` to embed an agent in any Go program.

```go
import "github.com/goppydae/gollm/sdk"

ag, err := sdk.NewAgent(sdk.Config{
    Provider: "ollama",
    Model:    "llama3.2",
    Tools:    sdk.DefaultTools(),
})
if err != nil {
    panic(err)
}

ag.Subscribe(func(e sdk.Event) {
    if e.Type == sdk.EventTextDelta {
        fmt.Print(e.Content)
    }
})

ag.Prompt(context.Background(), "List the Go files in this directory")
<-ag.Idle()
```

---

## Config Fields

```go
type Config struct {
    Provider    string        // "ollama", "openai", "anthropic", "llamacpp", "google"
    Model       string        // model name or "provider/model"
    APIKey      string        // optional; env vars take priority
    BaseURL     string        // optional provider endpoint override
    Tools       []sdk.Tool    // sdk.DefaultTools() or custom list
    Extensions  []sdk.Extension
    SystemPrompt string
    ThinkingLevel sdk.ThinkingLevel
    SessionDir  string        // where to persist sessions
    DryRun      bool
}
```

---

## Core API

| Call | Description |
|---|---|
| `sdk.NewAgent(cfg)` | Create and initialize an agent |
| `ag.Subscribe(fn)` | Register an event handler; called for every emitted event |
| `ag.Prompt(ctx, text)` | Send a user message and start the agent loop |
| `ag.Idle()` | Returns a channel that closes when the agent reaches Idle state |
| `ag.Steer(ctx, text)` | Inject a steering message into the running turn |
| `ag.FollowUp(ctx, text)` | Queue a message to process after the current turn |
| `ag.Abort(ctx)` | Cancel the current running turn |
| `ag.SetExtensions(exts)` | Replace the extension list (takes effect on next prompt) |

---

## Event Types

Subscribe to events by checking `e.Type`:

| Event type | Payload field | Description |
|---|---|---|
| `EventAgentStart` | — | Agent loop started |
| `EventAgentEnd` | — | Agent loop completed |
| `EventTurnStart` | — | LLM turn started |
| `EventTurnEnd` | — | LLM turn completed |
| `EventTextDelta` | `e.Content` | Incremental response text |
| `EventThinkingDelta` | `e.Content` | Incremental thinking text |
| `EventToolCall` | `e.ToolCall` | Tool invocation started |
| `EventToolDelta` | `e.Content` | Streaming tool output |
| `EventToolOutput` | `e.ToolOutput` | Final tool result |

---

## Minimal Example (no tools, no session)

```go
ag, _ := sdk.NewAgent(sdk.Config{
    Provider: "anthropic",
    Model:    "claude-sonnet-4-6",
    APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
})

var buf strings.Builder
ag.Subscribe(func(e sdk.Event) {
    if e.Type == sdk.EventTextDelta {
        buf.WriteString(e.Content)
    }
})

ag.Prompt(context.Background(), "What is 2+2?")
<-ag.Idle()
fmt.Println(buf.String())
```
