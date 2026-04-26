package extensions

import (
	"context"
	"encoding/json"
)

// ToolCall describes a tool invocation passed to Plugin hook methods.
type ToolCall struct {
	Name string
	Args json.RawMessage
}

// ToolResult is the outcome of a tool call or an interception.
type ToolResult struct {
	Content string
	IsError bool
}

// AgentState is the mutable prompt state passed to BeforePrompt.
type AgentState struct {
	SystemPrompt  string
	Model         string
	Provider      string
	ThinkingLevel string
}

// ToolDefinition describes a tool contributed by a Plugin.
type ToolDefinition struct {
	Name        string
	Description string
	Schema      json.RawMessage
	IsReadOnly  bool
}

// Plugin is the interface that standalone gRPC extension binaries implement.
// Embed NoopPlugin and override only the methods you need.
type Plugin interface {
	Name() string
	Tools() []ToolDefinition
	ExecuteTool(ctx context.Context, name string, args json.RawMessage) ToolResult
	BeforePrompt(ctx context.Context, state AgentState) AgentState
	BeforeToolCall(ctx context.Context, call ToolCall, args json.RawMessage) (ToolResult, bool)
	AfterToolCall(ctx context.Context, call ToolCall, result ToolResult) ToolResult
	ModifySystemPrompt(prompt string) string
}

// Compile-time check.
var _ Plugin = (*NoopPlugin)(nil)

// NoopPlugin is a base Plugin implementation with no-op defaults.
// Embed it in your Plugin struct and override only what you need.
type NoopPlugin struct {
	NameStr string
}

func (n *NoopPlugin) Name() string             { return n.NameStr }
func (n *NoopPlugin) Tools() []ToolDefinition  { return nil }
func (n *NoopPlugin) ExecuteTool(_ context.Context, name string, _ json.RawMessage) ToolResult {
	return ToolResult{Content: "tool not found: " + name, IsError: true}
}
func (n *NoopPlugin) BeforePrompt(_ context.Context, state AgentState) AgentState { return state }
func (n *NoopPlugin) BeforeToolCall(_ context.Context, _ ToolCall, _ json.RawMessage) (ToolResult, bool) {
	return ToolResult{}, false
}
func (n *NoopPlugin) AfterToolCall(_ context.Context, _ ToolCall, result ToolResult) ToolResult {
	return result
}
func (n *NoopPlugin) ModifySystemPrompt(prompt string) string { return prompt }
