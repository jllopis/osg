package summary

import (
	"context"
	"fmt"
	"strings"

	"github.com/jllopis/kairos/pkg/llm"
	"github.com/jllopis/kairos/providers/anthropic"
	"github.com/jllopis/kairos/providers/gemini"
	"github.com/jllopis/kairos/providers/openai"
	"github.com/jllopis/kairos/providers/qwen"
)

// DefaultSystemPromptTemplate is the default system instruction for AI summary
// generation when no custom prompt is configured.  The %s placeholder is
// replaced with a language clause (e.g. "in Spanish") when a language is set.
const DefaultSystemPromptTemplate = "Summarize the following blog post in 2-3 concise sentences%s for use as a preview excerpt. Return only the summary text, no labels or prefixes."

// buildDefaultPrompt returns the default system prompt, optionally injecting
// a language instruction.  When lang is empty the prompt is language-neutral.
func buildDefaultPrompt(lang string) string {
	if lang == "" {
		return fmt.Sprintf(DefaultSystemPromptTemplate, "")
	}
	name := langDisplayName(lang)
	return fmt.Sprintf(DefaultSystemPromptTemplate, " in "+name)
}

// langDisplayName maps a BCP-47 code to its English display name for prompt
// injection.  Only the most common codes are mapped; unknown codes are passed
// through as-is so the LLM can still interpret them.
func langDisplayName(code string) string {
	names := map[string]string{
		"es": "Spanish", "en": "English", "fr": "French",
		"de": "German", "it": "Italian", "pt": "Portuguese",
		"ca": "Catalan", "eu": "Basque", "gl": "Galician",
		"nl": "Dutch", "ja": "Japanese", "zh": "Chinese",
		"ko": "Korean", "ru": "Russian", "ar": "Arabic",
		"pl": "Polish", "sv": "Swedish", "da": "Danish",
		"fi": "Finnish", "no": "Norwegian", "tr": "Turkish",
	}
	if name, ok := names[strings.ToLower(code)]; ok {
		return name
	}
	return code
}

// KairosProvider generates summaries using an LLM via Kairos.
//
// It implements the Provider interface and delegates to a Kairos
// llm.Provider for the actual LLM call.  The provider is safe for
// concurrent use (Kairos providers are goroutine-safe).
type KairosProvider struct {
	// LLM is the underlying Kairos provider.
	LLM llm.Provider
	// Model overrides the provider's default model per-request.
	// If empty the provider's configured default is used.
	Model string
	// SystemPrompt is the system instruction sent to the LLM.
	// When empty a language-aware default is built from Language.
	SystemPrompt string
	// Language is the BCP-47 language code (e.g. "es") used to build
	// the default system prompt.  Ignored when SystemPrompt is set.
	Language string
}

// Summarize sends the page content to the LLM and returns the generated
// summary.  The context carries cancellation and timeout.
func (k *KairosProvider) Summarize(ctx context.Context, title string, rawMarkdown string) (string, error) {
	if k.LLM == nil {
		return "", fmt.Errorf("kairos: LLM provider is nil")
	}

	// Strip markdown formatting before sending to the LLM to reduce
	// token usage and give the model cleaner input.
	plain := PlainText(rawMarkdown)
	if plain == "" {
		return "", nil
	}

	prompt := k.SystemPrompt
	if prompt == "" {
		prompt = buildDefaultPrompt(k.Language)
	}

	userContent := fmt.Sprintf("Title: %s\n\n%s", title, plain)

	resp, err := k.LLM.Chat(ctx, llm.ChatRequest{
		Model: k.Model,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: prompt},
			{Role: llm.RoleUser, Content: userContent},
		},
		Temperature: 0.3,
	})
	if err != nil {
		return "", fmt.Errorf("kairos: %w", err)
	}

	return strings.TrimSpace(resp.Content), nil
}

// AIConfig holds the parameters needed to create a KairosProvider.
// This mirrors config.AIConfig but avoids importing the config package.
type AIConfig struct {
	Provider     string
	Model        string
	APIKey       string
	BaseURL      string
	SystemPrompt string
	// Language is the BCP-47 language code (e.g. "es") injected into the
	// default system prompt.  Ignored when SystemPrompt is set explicitly.
	Language string
}

// NewKairosProvider creates a KairosProvider from the given configuration.
// The ctx is required by some providers (e.g. gemini) at construction time.
func NewKairosProvider(ctx context.Context, cfg AIConfig) (*KairosProvider, error) {
	var provider llm.Provider
	var err error

	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "gemini":
		provider, err = newGeminiProvider(ctx, cfg)
	case "anthropic":
		provider, err = newAnthropicProvider(cfg)
	case "openai":
		provider, err = newOpenAIProvider(cfg)
	case "qwen":
		provider, err = newQwenProvider(cfg)
	case "ollama":
		provider, err = newOllamaProvider(cfg)
	default:
		return nil, fmt.Errorf("kairos: unknown provider %q", cfg.Provider)
	}
	if err != nil {
		return nil, err
	}

	return &KairosProvider{
		LLM:          provider,
		Model:        cfg.Model,
		SystemPrompt: cfg.SystemPrompt,
		Language:     cfg.Language,
	}, nil
}

func newGeminiProvider(ctx context.Context, cfg AIConfig) (llm.Provider, error) {
	var opts []gemini.Option
	if cfg.Model != "" {
		opts = append(opts, gemini.WithModel(cfg.Model))
	}
	if cfg.APIKey != "" {
		return gemini.NewWithAPIKey(ctx, cfg.APIKey, opts...)
	}
	return gemini.New(ctx, opts...)
}

func newAnthropicProvider(cfg AIConfig) (llm.Provider, error) {
	var opts []anthropic.Option
	if cfg.Model != "" {
		opts = append(opts, anthropic.WithModel(cfg.Model))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(cfg.BaseURL))
	}
	if cfg.APIKey != "" {
		opts = append(opts, anthropic.WithAPIKey(cfg.APIKey))
		return anthropic.New(opts...), nil
	}
	return anthropic.New(opts...), nil
}

func newOpenAIProvider(cfg AIConfig) (llm.Provider, error) {
	var opts []openai.Option
	if cfg.Model != "" {
		opts = append(opts, openai.WithModel(cfg.Model))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, openai.WithBaseURL(cfg.BaseURL))
	}
	if cfg.APIKey != "" {
		return openai.NewWithAPIKey(cfg.APIKey, opts...), nil
	}
	return openai.New(opts...), nil
}

func newQwenProvider(cfg AIConfig) (llm.Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("kairos: qwen provider requires an api_key (set ai.api_key or DASHSCOPE_API_KEY)")
	}
	var opts []qwen.Option
	if cfg.Model != "" {
		opts = append(opts, qwen.WithModel(cfg.Model))
	}
	if cfg.BaseURL != "" {
		opts = append(opts, qwen.WithBaseURL(cfg.BaseURL))
	}
	return qwen.New(cfg.APIKey, opts...), nil
}

func newOllamaProvider(cfg AIConfig) (llm.Provider, error) {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return llm.NewOllama(baseURL), nil
}
