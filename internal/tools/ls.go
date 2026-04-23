package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Ls is a tool for listing directory contents.
type Ls struct{}

func (Ls) Name() string { return "ls" }

func (Ls) Description() string {
	return "List files and directories in a given path."
}

func (Ls) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Path to list. Defaults to current directory."
			},
			"recursive": {
				"type": "boolean",
				"description": "List recursively. Defaults to false."
			},
			"long": {
				"type": "boolean",
				"description": "Use long format. Defaults to false."
			}
		}
	}`)
}

func (Ls) Execute(ctx context.Context, args json.RawMessage, update ToolUpdate) (*ToolResult, error) {
	var params struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
		Long      bool   `json:"long"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	path := params.Path
	if path == "" {
		path = "."
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	var entries []string
	if params.Recursive {
		_ = filepath.Walk(absPath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			rel, _ := filepath.Rel(absPath, p)
			if rel == "." {
				return nil
			}
			entries = append(entries, rel)
			return nil
		})
	} else {
		dir, err := os.ReadDir(absPath)
		if err != nil {
			return nil, fmt.Errorf("read directory: %w", err)
		}
		for _, d := range dir {
			entries = append(entries, d.Name())
		}
	}

	if update != nil {
		update(&ToolResult{
			Content: strings.Join(entries, "\n"),
			Metadata: map[string]any{
				"path":  absPath,
				"count": len(entries),
			},
		})
	}

	return &ToolResult{
		Content: strings.Join(entries, "\n"),
		Metadata: map[string]any{
			"path":  absPath,
			"count": len(entries),
		},
	}, nil
}
