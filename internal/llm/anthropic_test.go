package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goppydae/sharur/internal/types"
)

// sseLines returns an SSE response body from data lines.
func sseLines(lines ...string) string {
	var sb strings.Builder
	for _, l := range lines {
		fmt.Fprintf(&sb, "data: %s\n\n", l)
	}
	return sb.String()
}

func anthropicDeltaEvent(text string) string {
	return fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":%q}}`, text)
}

func anthropicMessageStart() string {
	return `{"type":"message_start","message":{"usage":{"input_tokens":10,"output_tokens":0}}}`
}

func anthropicContentBlockStart() string {
	return `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`
}

func anthropicMessageStop() string {
	return `{"type":"message_stop"}`
}

func TestAnthropicProvider_Stream_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		body := sseLines(
			anthropicMessageStart(),
			anthropicContentBlockStart(),
			anthropicDeltaEvent("hello"),
			anthropicDeltaEvent(" world"),
			anthropicMessageStop(),
		)
		_, _ = fmt.Fprint(w, body)
	}))
	defer srv.Close()

	p := NewAnthropicProvider("test-key", "claude-test").WithBaseURL(srv.URL)
	ch, err := p.Stream(context.Background(), &CompletionRequest{
		Model:    "claude-test",
		Messages: []types.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}

	var got strings.Builder
	for ev := range ch {
		if ev.Error != nil {
			t.Fatalf("unexpected event error: %v", ev.Error)
		}
		if ev.Type == EventTextDelta {
			got.WriteString(ev.Content)
		}
	}
	if got.String() != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", got.String())
	}
}

func TestAnthropicProvider_Stream_MalformedLine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Include a malformed JSON line — should not crash, just skip.
		body := sseLines(
			anthropicMessageStart(),
			anthropicContentBlockStart(),
			"this-is-not-json",
			anthropicDeltaEvent("ok"),
			anthropicMessageStop(),
		)
		_, _ = fmt.Fprint(w, body)
	}))
	defer srv.Close()

	p := NewAnthropicProvider("test-key", "claude-test").WithBaseURL(srv.URL)
	ch, err := p.Stream(context.Background(), &CompletionRequest{
		Model:    "claude-test",
		Messages: []types.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}

	var textDeltaCount int
	for ev := range ch {
		if ev.Error != nil {
			t.Fatalf("unexpected event error: %v", ev.Error)
		}
		if ev.Type == EventTextDelta {
			textDeltaCount++
		}
	}
	// The valid delta event ("ok") should still arrive.
	if textDeltaCount == 0 {
		t.Error("expected at least one text delta event despite malformed line")
	}
}

func TestAnthropicProvider_Stream_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"type":"error","error":{"type":"invalid_request_error","message":"bad request"}}`, http.StatusBadRequest)
	}))
	defer srv.Close()

	p := NewAnthropicProvider("test-key", "claude-test").WithBaseURL(srv.URL)
	ch, err := p.Stream(context.Background(), &CompletionRequest{
		Model:    "claude-test",
		Messages: []types.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Stream() returned error at call time (not expected): %v", err)
	}

	var gotErr bool
	for ev := range ch {
		if ev.Error != nil {
			gotErr = true
		}
	}
	if !gotErr {
		t.Error("expected an error event for 400 response, got none")
	}
}

func TestAnthropicProvider_Stream_ContextCancellation(t *testing.T) {
	// Server sends an infinite stream; the client should stop when the context is cancelled.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Block until the client disconnects (request context done).
		<-r.Context().Done()
	}))
	// Do NOT defer srv.Close() here — we close it after cancel to unblock the handler.

	ctx, cancel := context.WithCancel(context.Background())
	p := NewAnthropicProvider("test-key", "claude-test").WithBaseURL(srv.URL)
	ch, err := p.Stream(ctx, &CompletionRequest{
		Model:    "claude-test",
		Messages: []types.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		srv.Close()
		t.Fatalf("Stream() error: %v", err)
	}

	cancel()    // cancel the context, which unblocks the server handler
	srv.Close() // now the server can shut down cleanly

	// Drain — the channel must close without hanging.
	for range ch {
	}
}
