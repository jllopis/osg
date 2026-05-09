package summary

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jllopis/kairos/pkg/llm"
)

// ollamaProvider is a drop-in replacement for kairos's
// llm.OllamaProvider. The kairos version hardcodes a 120s HTTP client
// timeout that we cannot override from outside, which silently caps
// any longer cfg.AI.Timeout the operator configures (the build still
// wraps the call in context.WithTimeout, but the HTTP client cancels
// first). This implementation uses an http.Client without an
// internal Timeout, so the only deadline that matters is the one
// the caller pushes into the context — which is what cfg.AI.Timeout
// already feeds via internal/build/build.go.
//
// The wire protocol is identical to Ollama's POST /api/chat with
// stream=false, so any model that works with kairos's provider
// keeps working through this one.
type ollamaProvider struct {
	baseURL string
	client  *http.Client
}

// newOllamaClient constructs an ollamaProvider pointed at the given
// base URL. baseURL falls back to http://localhost:11434 when empty
// to match kairos's default.
func newOllamaClient(baseURL string) *ollamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &ollamaProvider{
		baseURL: baseURL,
		// Timeout: 0 -> rely entirely on the request context. The
		// caller (build.fillWithAI / ui.generateSummaryNow) wraps
		// every call with context.WithTimeout(cfg.AI.Timeout).
		client: &http.Client{},
	}
}

type ollamaChatRequest struct {
	Model    string                 `json:"model"`
	Messages []llm.Message          `json:"messages"`
	Stream   bool                   `json:"stream"`
	Tools    []llm.Tool             `json:"tools,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type ollamaChatResponse struct {
	Message         llm.Message `json:"message"`
	Done            bool        `json:"done"`
	TotalDuration   int64       `json:"total_duration"`
	EvalCount       int         `json:"eval_count"`
	PromptEvalCount int         `json:"prompt_eval_count"`
}

// Chat satisfies llm.Provider.
func (p *ollamaProvider) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	body := ollamaChatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
		Tools:    req.Tools,
	}
	if req.Temperature != 0 {
		body.Options = map[string]interface{}{"temperature": req.Temperature}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/chat", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ollama: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: api call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama: api returned status %d", resp.StatusCode)
	}

	var out ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("ollama: decode response: %w", err)
	}

	return &llm.ChatResponse{
		Content:   out.Message.Content,
		ToolCalls: out.Message.ToolCalls,
		Usage: llm.Usage{
			PromptTokens:     out.PromptEvalCount,
			CompletionTokens: out.EvalCount,
			TotalTokens:      out.PromptEvalCount + out.EvalCount,
		},
	}, nil
}
