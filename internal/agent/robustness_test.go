package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/goppydae/gollm/internal/llm"
	"github.com/goppydae/gollm/internal/tools"
)

type mockProvider struct {
	responses []*llm.Event
}

func (m *mockProvider) Stream(ctx context.Context, req *llm.CompletionRequest) (<-chan *llm.Event, error) {
	ch := make(chan *llm.Event, len(m.responses)+1)
	for _, r := range m.responses {
		ch <- r
	}
	close(ch)
	return ch, nil
}

func (m *mockProvider) Info() llm.ProviderInfo {
	return llm.ProviderInfo{Name: "mock", Model: "test"}
}

type mockTool struct {
	name     string
	readOnly bool
	called   bool
}

func (m *mockTool) Name() string                     { return m.name }
func (m *mockTool) Description() string              { return "mock" }
func (m *mockTool) Schema() json.RawMessage          { return json.RawMessage("{}") }
func (m *mockTool) IsReadOnly() bool                 { return m.readOnly }
func (m *mockTool) Execute(ctx context.Context, args json.RawMessage, update tools.ToolUpdate) (*tools.ToolResult, error) {
	m.called = true
	return &tools.ToolResult{Content: "done"}, nil
}

func TestMaxSteps(t *testing.T) {
	// Provider that always calls a tool
	prov := &mockProvider{
		responses: []*llm.Event{
			{Type: llm.EventToolCall, ToolCall: &llm.ToolCall{ID: "1", Name: "test", Args: json.RawMessage("{}")}},
		},
	}
	reg := tools.NewToolRegistry()
	reg.Register(&mockTool{name: "test"})

	ag := New(prov, reg)
	ag.SetMaxSteps(1)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	ag.Subscribe(func(e Event) {
		if e.Type == EventError {
			errCh <- e.Error
		}
	})

	err := ag.Prompt(ctx, "test")
	if err != nil {
		t.Fatalf("Prompt failed: %v", err)
	}

	select {
	case lastErr := <-errCh:
		if lastErr.Error() != "maximum recursive steps (1) exceeded" {
			t.Errorf("Unexpected error: %v", lastErr)
		}
	case <-ctx.Done():
		t.Fatal("Test timed out before receiving error")
	}

	// Also verify it goes idle eventually
	select {
	case <-ag.Idle():
		// ok
	case <-time.After(1 * time.Second):
		t.Fatal("Agent did not go idle after error")
	}
}

func TestDryRun(t *testing.T) {
	prov := &mockProvider{
		responses: []*llm.Event{
			{Type: llm.EventToolCall, ToolCall: &llm.ToolCall{ID: "1", Name: "write", Args: json.RawMessage("{}")}},
			{Type: llm.EventToolCall, ToolCall: &llm.ToolCall{ID: "2", Name: "read", Args: json.RawMessage("{}")}},
		},
	}
	reg := tools.NewToolRegistry()
	write := &mockTool{name: "write", readOnly: false}
	read := &mockTool{name: "read", readOnly: true}
	reg.Register(write)
	reg.Register(read)

	ag := New(prov, reg)
	ag.SetDryRun(true)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_ = ag.Prompt(ctx, "test")
	<-ag.Idle()

	if write.called {
		t.Error("Write tool was called in DryRun mode")
	}
	if !read.called {
		t.Error("Read tool was NOT called in DryRun mode")
	}
}
