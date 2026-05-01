package main

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/goppydae/sharur/extensions"
)

type sandboxPlugin struct {
	extensions.NoopPlugin
	root string
}

func newSandbox(dir string) (*sandboxPlugin, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve sandbox root: %w", err)
	}
	return &sandboxPlugin{
		NoopPlugin: extensions.NoopPlugin{NameStr: "sandbox"},
		root:       abs,
	}, nil
}

func (s *sandboxPlugin) BeforeToolCall(_ context.Context, call extensions.ToolCall, args json.RawMessage) (extensions.ToolResult, bool) {
	keys := pathArgKeys(call.Name)
	if len(keys) == 0 {
		return extensions.ToolResult{}, false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(args, &m); err != nil {
		return extensions.ToolResult{}, false
	}
	for _, key := range keys {
		raw, ok := m[key]
		if !ok {
			continue
		}
		var val string
		if err := json.Unmarshal(raw, &val); err != nil || val == "" {
			continue
		}
		val = strings.TrimPrefix(val, "@") // strip sharur's '@' path prefix
		abs, err := filepath.Abs(val)
		if err != nil {
			return extensions.ToolResult{
				Content: fmt.Sprintf("sandbox: cannot resolve path %q: %v", val, err),
				IsError: true,
			}, true
		}
		if !withinRoot(abs, s.root) {
			return extensions.ToolResult{
				Content: fmt.Sprintf("sandbox: path %q is outside allowed root %q", abs, s.root),
				IsError: true,
			}, true
		}
	}
	return extensions.ToolResult{}, false
}

func (s *sandboxPlugin) ModifySystemPrompt(prompt string) string {
	return prompt + fmt.Sprintf(
		"\n\n<sandbox>All file operations are restricted to: %s\n"+
			"Paths outside this directory will be rejected.</sandbox>",
		s.root)
}

func pathArgKeys(toolName string) []string {
	switch toolName {
	case "read", "write", "edit", "ls", "find":
		return []string{"path"}
	case "grep":
		return []string{"root"}
	case "bash":
		return []string{"cwd"}
	default:
		return nil
	}
}

func withinRoot(abs, root string) bool {
	return abs == root || strings.HasPrefix(abs, root+string(filepath.Separator))
}
