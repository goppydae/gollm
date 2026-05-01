package llm

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// LlamaCppProvider implements the Provider interface for a llama.cpp server.
// It reuses OpenAIProvider for streaming (llama.cpp exposes /v1/chat/completions)
// and discovers the context window size from the server's /props endpoint.
type LlamaCppProvider struct {
	*OpenAIProvider
	nCtx int
}

// NewLlamaCppProvider creates a provider pointed at a llama.cpp server.
// baseURL defaults to http://localhost:8080.
// Model name and context window are discovered from the server on construction.
func NewLlamaCppProvider(baseURL string) *LlamaCppProvider {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	base := NewOpenAIProvider(baseURL, "llamacpp")
	p := &LlamaCppProvider{OpenAIProvider: base}
	if name := p.fetchModelName(); name != "" {
		p.model = name
	}
	p.nCtx = p.fetchNCtx()
	return p
}

func (p *LlamaCppProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:          "llamacpp",
		Model:         p.model,
		MaxTokens:     p.maxTokens,
		ContextWindow: p.nCtx,
		HasToolCall:   true,
		HasImages:     false,
	}
}

// fetchNCtx queries /props and returns n_ctx, or 0 on failure.
func (p *LlamaCppProvider) fetchNCtx() int {
	discoveryClient := &http.Client{Timeout: 2 * time.Second}
	resp, err := discoveryClient.Get(p.baseURL + "/props")
	if err != nil {
		return 0
	}
	defer func() { _ = resp.Body.Close() }()

	var props struct {
		NCtx                      int `json:"n_ctx"`
		DefaultGenerationSettings struct {
			NCtx int `json:"n_ctx"`
		} `json:"default_generation_settings"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&props); err != nil {
		return 0
	}
	if props.NCtx > 0 {
		return props.NCtx
	}
	if props.DefaultGenerationSettings.NCtx > 0 {
		return props.DefaultGenerationSettings.NCtx
	}
	// If we got a valid response but couldn't find n_ctx, fallback to a sensible default
	return 4096
}

// fetchModelName queries /v1/models and returns the first model ID, or "".
func (p *LlamaCppProvider) fetchModelName() string {
	discoveryClient := &http.Client{Timeout: 2 * time.Second}
	resp, err := discoveryClient.Get(p.baseURL + "/v1/models")
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	if len(result.Data) == 0 {
		return ""
	}
	return cleanModelName(result.Data[0].ID)
}

// cleanModelName strips directory path and file extension from raw model IDs
// that llama.cpp sometimes returns as full file paths.
func cleanModelName(raw string) string {
	name := filepath.Base(raw)
	if ext := filepath.Ext(name); ext != "" {
		name = strings.TrimSuffix(name, ext)
	}
	return name
}

func (p *LlamaCppProvider) ListModels() ([]string, error) {
	name := p.fetchModelName()
	if name == "" {
		return nil, nil
	}
	return []string{name}, nil
}

var _ Provider = (*LlamaCppProvider)(nil)
var _ ModelLister = (*LlamaCppProvider)(nil)
