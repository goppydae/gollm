---
title: LLM Providers
weight: 30
description: Provider interface, CompletionRequest, ProviderInfo, ModelLister, and per-adapter notes
---

## Provider Interface

```go
type Provider interface {
    Stream(ctx context.Context, req *CompletionRequest) (<-chan *Event, error)
    Info() ProviderInfo
}
```

All providers return a uniform `Stream` of `Event` values — text deltas, thinking deltas, tool calls, and usage. The agent's `consumeStream` function normalizes these into the internal `Message` format, making the agent completely provider-agnostic.

---

## CompletionRequest

```go
type CompletionRequest struct {
    Model       string
    Messages    []types.Message
    Tools       []types.ToolInfo
    System      string
    Thinking    types.ThinkingLevel
    MaxTokens   int
    Temperature float64
    StreamOpts  StreamOptions
}
```

The `BeforeProviderRequest` extension hook receives this struct as JSON and can modify any field before it is sent to the provider — useful for overriding temperature, trimming the tool list, or adjusting `MaxTokens` per request.

---

## ProviderInfo

```go
type ProviderInfo struct {
    Name          string
    Model         string
    MaxTokens     int
    ContextWindow int  // 0 = unknown
    HasToolCall   bool
    HasImages     bool
}
```

`Info()` is called once at startup. The service uses `ContextWindow` to trigger compaction when the conversation grows too large. `HasImages` controls whether the TUI offers image attachment UI.

---

## ModelLister

```go
type ModelLister interface {
    ListModels() ([]string, error)
}
```

All five adapters implement `ModelLister`. When `--list-models` is passed, the CLI casts the active provider to `ModelLister` and prints the result. Each adapter queries the appropriate API:

| Provider | Query mechanism |
|---|---|
| `ollama` | `GET /api/tags` |
| `llamacpp` | `GET /v1/models` |
| `openai` | `GET /v1/models` |
| `anthropic` | `GET /v1/models` |
| `google` | Gemini model list API |

---

## Supported Providers

| Provider | Backend |
|---|---|
| `ollama` | Local Ollama server (HTTP) |
| `llamacpp` | llama.cpp server (HTTP, OpenAI-compatible) |
| `openai` | OpenAI API or any OpenAI-compatible endpoint |
| `anthropic` | Anthropic Messages API |
| `google` | Google Gemini API |

Each adapter lives in `internal/llm/` and translates the provider's wire format into the uniform `Stream` abstraction.

---

## Feature Matrix

| Provider | Tools | Images | Thinking | Context Window |
|---|:---:|:---:|:---:|---|
| `ollama` | ✓ | ✓ | model-dependent | 4096 (default) |
| `llamacpp` | ✓ | ✗ | ✗ | from server `n_ctx` |
| `openai` | ✓ | ✓ | reasoning models | model-dependent |
| `anthropic` | ✓ | ✓ | ✓ extended | model-dependent |
| `google` | ✓ | ✓ | ✗ | 1,000,000+ |

---

## Per-Provider Notes

### Ollama

The Ollama adapter uses the `/api/chat` endpoint with streaming enabled. Context window defaults to 4096 when not reported by the server. Thinking is supported on models that emit `<think>` tokens (e.g. `qwq`, `deepseek-r1`) — `gollm` surfaces these as `EventThinkingDelta` events by detecting the tag boundaries in the stream.

### llama.cpp

Uses the OpenAI-compatible `/v1/chat/completions` endpoint. The context window (`n_ctx`) is queried from the server at startup. Image attachments are not supported because llama.cpp's OpenAI endpoint does not accept multipart vision payloads in the standard format.

### OpenAI

Uses the standard `/v1/chat/completions` streaming endpoint. Any server implementing this API — vLLM, LM Studio, Groq, Together AI — can be used by setting `openAIBaseURL`. Reasoning models (o3, o4-mini) emit `reasoning_content` deltas that are surfaced as `EventThinkingDelta`.

### Anthropic

Uses the Messages API (`/v1/messages`) with streaming. Extended thinking is activated when `req.Thinking` is `medium` or `high`:

- **medium** — 10,000-token thinking budget
- **high** — 20,000-token thinking budget

The API requires `temperature: 1.0` when extended thinking is enabled; the adapter sets this automatically and overrides any user-supplied temperature for that request.

### Google

Uses the Gemini `generateContent` API via the `google.golang.org/genai` client library. Gemini 1.5 Pro and later have context windows of 1M+ tokens; compaction is rarely triggered for typical sessions.

---

## Adding a Provider

Implement the `Provider` interface in `internal/llm/yourprovider.go` and register it in `internal/config/factory.go`. Implement `ModelLister` to enable `--list-models`. The adapter receives a fully-formed `CompletionRequest`; it is responsible for translating `Message.ToolCalls` and `Message.Images` into the target API's format.
