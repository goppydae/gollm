package llm

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
)

// StreamHTTP handles the boilerplate for HTTP streaming from an LLM provider.
func StreamHTTP(ctx context.Context, client *http.Client, req *http.Request, ch chan<- *Event, chunkHandler func(line string) error) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var body bytes.Buffer
		_, _ = body.ReadFrom(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body.String())
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()
		if err := chunkHandler(line); err != nil {
			// We don't necessarily want to abort on all chunk errors (e.g. unmarshal skip)
			// The handler should decide whether to return an error.
			continue 
		}
	}
	return scanner.Err()
}

// ModelContextWindows provides a central registry of known model context windows.
var ModelContextWindows = map[string]int{
	"gpt-4o":       128000,
	"gpt-4-turbo":  128000,
	"gpt-4":        8192,
	"gpt-3.5":      16385,
	"claude-3-5":   200000,
	"claude-3":     200000,
}

// GetContextWindow returns the context window for a model name, or 0 if unknown.
func GetContextWindow(model string) int {
	for k, v := range ModelContextWindows {
		if strings.HasPrefix(model, k) {
			return v
		}
	}
	return 0
}
