package summary

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jllopis/kairos/pkg/llm"
)

// ---------------------------------------------------------------------------
// KairosProvider.Summarize
// ---------------------------------------------------------------------------

func TestKairosProvider_BasicSummary(t *testing.T) {
	mock := &llm.MockProvider{Response: "This is a concise summary of the post."}
	kp := &KairosProvider{LLM: mock}

	text, err := kp.Summarize(context.Background(), "My Post", "Long markdown content here.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "This is a concise summary of the post." {
		t.Errorf("expected mock response, got %q", text)
	}
}

func TestKairosProvider_TrimsWhitespace(t *testing.T) {
	mock := &llm.MockProvider{Response: "  Summary with spaces.  \n"}
	kp := &KairosProvider{LLM: mock}

	text, err := kp.Summarize(context.Background(), "Title", "Content here.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Summary with spaces." {
		t.Errorf("expected trimmed response, got %q", text)
	}
}

func TestKairosProvider_EmptyContent(t *testing.T) {
	mock := &llm.MockProvider{Response: "Should not be called."}
	kp := &KairosProvider{LLM: mock}

	text, err := kp.Summarize(context.Background(), "Title", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty for empty content, got %q", text)
	}
}

func TestKairosProvider_WhitespaceOnlyContent(t *testing.T) {
	mock := &llm.MockProvider{Response: "Should not be called."}
	kp := &KairosProvider{LLM: mock}

	text, err := kp.Summarize(context.Background(), "Title", "   \n\n   ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty for whitespace-only content, got %q", text)
	}
}

func TestKairosProvider_LLMError(t *testing.T) {
	mock := &llm.FailingMockProvider{Err: fmt.Errorf("API rate limit exceeded")}
	kp := &KairosProvider{LLM: mock}

	_, err := kp.Summarize(context.Background(), "Title", "Some content.")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "kairos:") {
		t.Errorf("expected wrapped error with kairos prefix, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "API rate limit exceeded") {
		t.Errorf("expected original error message, got %q", err.Error())
	}
}

func TestKairosProvider_NilLLM(t *testing.T) {
	kp := &KairosProvider{LLM: nil}

	_, err := kp.Summarize(context.Background(), "Title", "Content.")
	if err == nil {
		t.Fatal("expected error for nil LLM, got nil")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected nil-related error, got %q", err.Error())
	}
}

func TestKairosProvider_CustomSystemPrompt(t *testing.T) {
	customPrompt := "Resumen en espanol, 1 frase."
	var capturedReq llm.ChatRequest

	mock := &llm.MockProvider{
		ChatFunc: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			capturedReq = req
			return &llm.ChatResponse{Content: "Resumen personalizado."}, nil
		},
	}
	kp := &KairosProvider{LLM: mock, SystemPrompt: customPrompt}

	text, err := kp.Summarize(context.Background(), "Mi Post", "Contenido del post.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Resumen personalizado." {
		t.Errorf("got %q", text)
	}

	// Verify the system prompt was used.
	if len(capturedReq.Messages) < 1 {
		t.Fatal("expected at least 1 message")
	}
	if capturedReq.Messages[0].Content != customPrompt {
		t.Errorf("expected custom prompt %q, got %q", customPrompt, capturedReq.Messages[0].Content)
	}
}

func TestKairosProvider_DefaultSystemPrompt(t *testing.T) {
	var capturedReq llm.ChatRequest

	mock := &llm.MockProvider{
		ChatFunc: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			capturedReq = req
			return &llm.ChatResponse{Content: "Summary."}, nil
		},
	}
	kp := &KairosProvider{LLM: mock} // no custom prompt

	_, err := kp.Summarize(context.Background(), "Title", "Content.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedReq.Messages[0].Content != DefaultSystemPrompt {
		t.Errorf("expected default prompt, got %q", capturedReq.Messages[0].Content)
	}
}

func TestKairosProvider_ModelOverride(t *testing.T) {
	var capturedReq llm.ChatRequest

	mock := &llm.MockProvider{
		ChatFunc: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			capturedReq = req
			return &llm.ChatResponse{Content: "Summary."}, nil
		},
	}
	kp := &KairosProvider{LLM: mock, Model: "custom-model-v2"}

	_, err := kp.Summarize(context.Background(), "Title", "Content.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedReq.Model != "custom-model-v2" {
		t.Errorf("expected model %q, got %q", "custom-model-v2", capturedReq.Model)
	}
}

func TestKairosProvider_SendsPlainText(t *testing.T) {
	var capturedReq llm.ChatRequest

	mock := &llm.MockProvider{
		ChatFunc: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			capturedReq = req
			return &llm.ChatResponse{Content: "Summary."}, nil
		},
	}
	kp := &KairosProvider{LLM: mock}

	md := "# Heading\n\nThis is **bold** and has a [[wiki link]]."
	_, err := kp.Summarize(context.Background(), "Title", md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The user message should contain plain text, not markdown.
	userMsg := capturedReq.Messages[1].Content
	if strings.Contains(userMsg, "**") || strings.Contains(userMsg, "[[") || strings.Contains(userMsg, "#") {
		t.Errorf("user message still contains markdown: %q", userMsg)
	}
	if !strings.Contains(userMsg, "Title: Title") {
		t.Errorf("expected title in user message, got %q", userMsg)
	}
}

func TestKairosProvider_Temperature(t *testing.T) {
	var capturedReq llm.ChatRequest

	mock := &llm.MockProvider{
		ChatFunc: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			capturedReq = req
			return &llm.ChatResponse{Content: "Summary."}, nil
		},
	}
	kp := &KairosProvider{LLM: mock}

	_, err := kp.Summarize(context.Background(), "Title", "Content.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedReq.Temperature != 0.3 {
		t.Errorf("expected temperature 0.3, got %f", capturedReq.Temperature)
	}
}

func TestKairosProvider_ContextCancellation(t *testing.T) {
	mock := &llm.MockProvider{
		ChatFunc: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	kp := &KairosProvider{LLM: mock}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := kp.Summarize(ctx, "Title", "Content.")
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestKairosProvider_ConcurrentUse(t *testing.T) {
	var callCount atomic.Int32

	mock := &llm.MockProvider{
		ChatFunc: func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
			callCount.Add(1)
			time.Sleep(10 * time.Millisecond) // simulate API latency
			return &llm.ChatResponse{Content: "Summary."}, nil
		},
	}
	kp := &KairosProvider{LLM: mock}

	const n = 10
	errs := make(chan error, n)
	for range n {
		go func() {
			_, err := kp.Summarize(context.Background(), "Title", "Content.")
			errs <- err
		}()
	}

	for range n {
		if err := <-errs; err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	}

	if got := callCount.Load(); got != n {
		t.Errorf("expected %d calls, got %d", n, got)
	}
}

// ---------------------------------------------------------------------------
// NewKairosProvider (factory)
// ---------------------------------------------------------------------------

func TestNewKairosProvider_UnknownProvider(t *testing.T) {
	_, err := NewKairosProvider(context.Background(), AIConfig{Provider: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("expected 'unknown provider' in error, got %q", err.Error())
	}
}

func TestNewKairosProvider_QwenRequiresAPIKey(t *testing.T) {
	_, err := NewKairosProvider(context.Background(), AIConfig{Provider: "qwen"})
	if err == nil {
		t.Fatal("expected error for qwen without api_key")
	}
	if !strings.Contains(err.Error(), "api_key") {
		t.Errorf("expected api_key-related error, got %q", err.Error())
	}
}

func TestNewKairosProvider_OllamaDefaultBaseURL(t *testing.T) {
	kp, err := NewKairosProvider(context.Background(), AIConfig{Provider: "ollama"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kp == nil {
		t.Fatal("expected non-nil provider")
	}
	if kp.LLM == nil {
		t.Fatal("expected non-nil LLM")
	}
}

func TestNewKairosProvider_OllamaCustomBaseURL(t *testing.T) {
	kp, err := NewKairosProvider(context.Background(), AIConfig{
		Provider: "ollama",
		BaseURL:  "http://my-ollama:11434",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kp == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewKairosProvider_SetsModelAndPrompt(t *testing.T) {
	kp, err := NewKairosProvider(context.Background(), AIConfig{
		Provider:     "ollama",
		Model:        "llama3.2",
		SystemPrompt: "Custom prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kp.Model != "llama3.2" {
		t.Errorf("expected model %q, got %q", "llama3.2", kp.Model)
	}
	if kp.SystemPrompt != "Custom prompt" {
		t.Errorf("expected prompt %q, got %q", "Custom prompt", kp.SystemPrompt)
	}
}

// ---------------------------------------------------------------------------
// AIConfig validation edge cases
// ---------------------------------------------------------------------------

func TestNewKairosProvider_EmptyProviderName(t *testing.T) {
	_, err := NewKairosProvider(context.Background(), AIConfig{Provider: ""})
	if err == nil {
		t.Fatal("expected error for empty provider")
	}
}

func TestNewKairosProvider_ProviderCaseInsensitive(t *testing.T) {
	for _, name := range []string{"OLLAMA", "Ollama", " ollama "} {
		kp, err := NewKairosProvider(context.Background(), AIConfig{Provider: name})
		if err != nil {
			t.Errorf("NewKairosProvider(%q): unexpected error: %v", name, err)
		}
		if kp == nil {
			t.Errorf("NewKairosProvider(%q): expected non-nil", name)
		}
	}
}
