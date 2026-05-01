---
title: Events
weight: 30
description: EventBus, event types, and subscription patterns
categories: [sdk]
tags: [events]
---

The agent communicates state transitions via an event bus. Every meaningful action emits an `sdk.Event` to all registered subscribers.

---

## Subscribing

```go
ag.Subscribe(func(e sdk.Event) {
    switch e.Type {
    case sdk.EventTextDelta:
        fmt.Print(e.Content)
    case sdk.EventToolCall:
        fmt.Printf("[tool: %s]\n", e.ToolCall.Name)
    case sdk.EventAgentEnd:
        fmt.Println("\ndone")
    }
})
```

Multiple subscribers are allowed. Each runs in its own goroutine. The EventBus is non-blocking — `Publish` enqueues to a 4096-item buffered channel per subscriber and returns immediately, so slow subscribers drop events rather than stalling the agent loop.

---

## Event Reference

| Type constant | Payload | Fired when |
|---|---|---|
| `EventAgentStart` | — | `Prompt()` called, agent loop begins |
| `EventAgentEnd` | — | Agent loop completes (all turns done) |
| `EventTurnStart` | — | An LLM request turn begins |
| `EventTurnEnd` | — | A turn's tool calls finish |
| `EventMessageStart` | — | LLM starts streaming a response |
| `EventMessageEnd` | — | LLM response stream complete |
| `EventTextDelta` | `e.Content string` | Incremental response text chunk |
| `EventThinkingDelta` | `e.Content string` | Incremental extended-thinking chunk |
| `EventToolCall` | `e.ToolCall` | Tool invocation requested by LLM |
| `EventToolDelta` | `e.Content string` | Streaming partial output from a running tool |
| `EventToolOutput` | `e.ToolOutput` | Final tool result (success or error) |

---

## Event Flow Per Prompt

```
EventAgentStart
  EventTurnStart
    EventMessageStart
      EventTextDelta*
      EventThinkingDelta*
      EventToolCall*
    EventMessageEnd
    [per tool call]
      EventToolDelta*
      EventToolOutput
  EventTurnEnd
  [repeat if tool calls triggered another turn]
EventAgentEnd
```

---

## Agent State Machine

The agent transitions through explicit states visible via `EventAgentStart`/`EventAgentEnd` and the `ag.Idle()` channel:

```
Idle → Thinking → Executing → Idle
           ↓
       Compacting → Idle
           ↓
         Aborting → Idle
```

`ag.Idle()` returns a channel that closes when the agent returns to `Idle`. Use it to block until a prompt completes:

```go
ag.Prompt(ctx, "Refactor main.go")
<-ag.Idle()
// agent is idle, safe to call Prompt again
```
