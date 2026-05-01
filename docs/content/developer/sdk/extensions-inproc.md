---
title: In-Process Extensions
weight: 40
description: Implementing agent.Extension directly for zero-overhead in-process hooks
categories: [sdk, extensions]
---

If your extension is written in Go and you control the build, you can implement `sdk.Extension` (an alias of `agent.Extension`) directly — no gRPC, no subprocess, no socket. This is the lowest-overhead extension path.

---

## Attaching Extensions

```go
type loggingExt struct {
    sdk.NoopExtension
}

func (e *loggingExt) AgentStart(ctx context.Context) { log.Println("agent started") }
func (e *loggingExt) AgentEnd(ctx context.Context)   { log.Println("agent finished") }
func (e *loggingExt) ModifyInput(ctx context.Context, text string) sdk.InputResult {
    if text == "quit" {
        return sdk.InputResult{Action: sdk.InputHandled}
    }
    return sdk.InputResult{Action: sdk.InputContinue}
}

ag.SetExtensions([]sdk.Extension{
    &loggingExt{NoopExtension: sdk.NoopExtension{NameStr: "logger"}},
})
```

`sdk.NoopExtension` provides no-op defaults for every method. Embed it and override only what you need.

---

## Extension Interface

```go
type Extension interface {
    Name() string
    Tools() []Tool

    SessionStart(ctx context.Context, sessionID string, reason SessionStartReason)
    SessionEnd(ctx context.Context, sessionID string, reason SessionEndReason)

    AgentStart(ctx context.Context)
    AgentEnd(ctx context.Context)
    TurnStart(ctx context.Context)
    TurnEnd(ctx context.Context)

    ModifyInput(ctx context.Context, text string) InputResult
    ModifySystemPrompt(prompt string) string
    BeforePrompt(ctx context.Context, state *AgentState) *AgentState
    ModifyContext(ctx context.Context, messages []Message) []Message
    BeforeProviderRequest(ctx context.Context, req *CompletionRequest) *CompletionRequest
    AfterProviderResponse(ctx context.Context, content string, numToolCalls int)
    BeforeToolCall(ctx context.Context, call *ToolCall, args json.RawMessage) (*ToolResult, bool)
    AfterToolCall(ctx context.Context, call *ToolCall, result *ToolResult) *ToolResult
    BeforeCompact(ctx context.Context, prep CompactionPrep) *CompactionResult
    AfterCompact(ctx context.Context, freedTokens int)
}
```

All types are re-exported from `sdk` so callers only need to import `github.com/goppydae/gollm/sdk`.

---

## Key Hook Behaviours

**`ModifyInput`** — runs before the user text is added to the transcript. Return an `InputResult` with:
- `sdk.InputContinue` — pass through unchanged
- `sdk.InputTransform` — replace with `result.Text`
- `sdk.InputHandled` — consume entirely; no agent turn is started and nothing is appended to the transcript

**`ModifyContext`** — receives and returns the message slice that will be sent to the LLM. Changes do not affect the stored session transcript — they are ephemeral per-turn.

**`BeforeToolCall`** — return `(result, true)` to intercept and block the tool; return `(nil, false)` to allow normal execution.

**`BeforeCompact`** — return `nil` to let the default LLM summarization run, or a `*CompactionResult` to supply your own summary and skip the LLM call.

---

## Example: System Prompt Injection

```go
type gitContextExt struct {
    sdk.NoopExtension
}

func (e *gitContextExt) ModifySystemPrompt(prompt string) string {
    branch, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
    return prompt + "\n\nCurrent git branch: " + strings.TrimSpace(string(branch))
}
```

---

## Example: Tool Interception

```go
type sandboxExt struct {
    sdk.NoopExtension
    allowedDir string
}

func (e *sandboxExt) BeforeToolCall(_ context.Context, call *sdk.ToolCall, args json.RawMessage) (*sdk.ToolResult, bool) {
    var input struct{ Path string `json:"path"` }
    _ = json.Unmarshal(args, &input)
    if input.Path != "" && !strings.HasPrefix(input.Path, e.allowedDir) {
        return &sdk.ToolResult{
            Content: fmt.Sprintf("blocked: %s is outside %s", input.Path, e.allowedDir),
            IsError: true,
        }, true
    }
    return nil, false
}
```
