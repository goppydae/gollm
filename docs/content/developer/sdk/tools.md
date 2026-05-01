---
title: Custom Tools
weight: 20
description: Implementing the Tool interface and registering tools with NewAgent
categories: [sdk]
tags: [tools]
---

## Built-in Tools

Pass `sdk.DefaultTools()` in `sdk.Config.Tools` to get the full set of built-in tools:

| Tool | Description |
|---|---|
| `read` | Read file contents with offset/limit support |
| `write` | Create or overwrite files |
| `edit` | Search-and-replace edits within files |
| `bash` | Execute shell commands |
| `grep` | Search file contents via regex |
| `ls` | List directory contents |
| `find` | Locate files using glob patterns |

> `bash`, `write`, and `edit` are destructive. In `--dry-run` mode they preview what they would do without executing.

---

## Tool Interface

Implement `sdk.Tool` to create a custom tool:

```go
type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage       // JSON Schema for the input parameters
    Execute(ctx context.Context, args json.RawMessage, update ToolUpdate) (*ToolResult, error)
    IsReadOnly() bool              // if true, tool is allowed in dry-run mode
}
```

`ToolUpdate` is a callback for streaming partial output while the tool runs:

```go
type ToolUpdate func(content string)
```

---

## Example: Custom Tool

```go
type CountLinesTool struct{}

func (t *CountLinesTool) Name() string { return "count_lines" }
func (t *CountLinesTool) Description() string {
    return "Count the number of lines in a file"
}
func (t *CountLinesTool) Schema() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "path": {"type": "string", "description": "File path to count lines in"}
        },
        "required": ["path"]
    }`)
}
func (t *CountLinesTool) IsReadOnly() bool { return true }

func (t *CountLinesTool) Execute(ctx context.Context, args json.RawMessage, update sdk.ToolUpdate) (*sdk.ToolResult, error) {
    var input struct {
        Path string `json:"path"`
    }
    if err := json.Unmarshal(args, &input); err != nil {
        return nil, err
    }
    data, err := os.ReadFile(input.Path)
    if err != nil {
        return &sdk.ToolResult{Content: err.Error(), IsError: true}, nil
    }
    n := strings.Count(string(data), "\n") + 1
    return &sdk.ToolResult{Content: fmt.Sprintf("%d lines", n)}, nil
}
```

Register alongside the built-in tools:

```go
ag, _ := sdk.NewAgent(sdk.Config{
    Provider: "ollama",
    Model:    "llama3.2",
    Tools:    append(sdk.DefaultTools(), &CountLinesTool{}),
})
```

---

## Selective Tools

Pass only the tools you want rather than the full default set:

```go
tools := sdk.ToolsFor("read", "grep", "ls")   // subset by name
```

Or build the list manually to include only read-only tools for a sandboxed agent.
