package summary

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/jllopis/kairos/pkg/llm"
	"github.com/jllopis/kairos/providers/anthropic"
	"github.com/jllopis/kairos/providers/gemini"
	"github.com/jllopis/kairos/providers/openai"
	"github.com/jllopis/kairos/providers/qwen"
)

// DefaultSystemPromptTemplate is the default system instruction for AI summary
// generation when no custom prompt is configured.  The %s placeholder is
// replaced with a language clause (e.g. "in Spanish") when a language is set.
//
// The prompt is deliberately strict and repetitive because small local models
// (e.g. Ollama) tend to produce verbose structured output if not firmly
// constrained.
const DefaultSystemPromptTemplate = `Write a single short paragraph (2-3 sentences, maximum 200 characters)%s summarizing the following blog post for use as a preview excerpt.

STRICT RULES:
- Output ONLY the summary paragraph, nothing else.
- NO titles, headings, labels, prefixes, or section numbers.
- NO bullet points, numbered lists, or structured formats.
- NO markdown formatting (no asterisks, underscores, hashes).
- The summary must be a single continuous paragraph of plain text.`

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

	// Truncate to avoid sending entire long articles to the LLM.
	// The first ~1500 words contain enough context for a good summary
	// and keep token usage low, which also prevents context-window
	// errors with local models (e.g. Ollama).
	plain = truncateWords(plain, maxInputWords)

	prompt := k.SystemPrompt
	if prompt == "" {
		prompt = buildDefaultPrompt(k.Language)
	}

	userContent := fmt.Sprintf("Title: %s\n\n%s", title, plain)

	req := llm.ChatRequest{
		Model: k.Model,
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: prompt},
			{Role: llm.RoleUser, Content: userContent},
		},
		Temperature: 0.3,
	}
	resp, err := chatWithRetry(ctx, k.LLM, req)
	if err != nil {
		return "", fmt.Errorf("kairos: %w", err)
	}

	// Strip any markdown formatting the LLM might have included and enforce
	// maximum length.  Small local models sometimes ignore length constraints
	// and produce verbose structured output.
	summary := PlainText(strings.TrimSpace(resp.Content))

	// Reject structured output (numbered lists, headings, labels) that
	// small models produce despite explicit instructions.  Fall back to
	// extracting the first sentences from the original content.
	if isStructuredOutput(summary) {
		summary = truncateSentence(PlainText(rawMarkdown), maxSummaryLen)
	} else {
		summary = truncateSentence(summary, maxSummaryLen)
	}
	return summary, nil
}

// retryAttempts and retryBackoff control the simple bounded retry
// policy applied to every LLM Chat call. Three attempts total with an
// exponential backoff (250ms, 500ms) cover most transient failures
// (timeouts, brief 5xx, 429 throttling) without dragging the build
// when the provider is genuinely down.
const (
	retryAttempts = 3
	retryBackoff  = 250 * time.Millisecond
)

// chatWithRetry wraps llm.Chat with a small retry loop. Errors are
// classified by isRetryable: auth/config/4xx-style failures fail fast,
// everything else (network, timeout, 5xx, 429) is retried up to
// retryAttempts-1 times. The caller's context is honoured between
// attempts so cancellation aborts the loop immediately.
func chatWithRetry(ctx context.Context, p llm.Provider, req llm.ChatRequest) (*llm.ChatResponse, error) {
	var (
		resp *llm.ChatResponse
		err  error
	)
	for attempt := 0; attempt < retryAttempts; attempt++ {
		if attempt > 0 {
			delay := retryBackoff * time.Duration(1<<(attempt-1))
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			slog.Default().Warn("kairos: retrying summary call", "attempt", attempt+1, "previous_error", err)
		}
		resp, err = p.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		if !isRetryable(err) {
			return nil, err
		}
	}
	return nil, err
}

// isRetryable classifies LLM errors. Anything that points at a
// permanent client-side problem (auth, malformed request, missing
// resource) fails fast — retrying would only burn time. Everything
// else, including timeouts and 5xx, is treated as transient.
//
// Kairos surfaces provider errors as plain strings without a typed
// status, so we lean on substring matching against well-known markers.
// False positives only mean we skip retries when retrying might have
// helped, never the inverse — so the policy stays conservative.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	msg := strings.ToLower(err.Error())
	nonRetryableMarkers := []string{
		"401", "unauthorized",
		"403", "forbidden",
		"400", "bad request", "invalid_request", "invalid request",
		"404", "not found",
		"invalid api key", "invalid_api_key",
		"permission denied",
	}
	for _, m := range nonRetryableMarkers {
		if strings.Contains(msg, m) {
			return false
		}
	}
	return true
}

// maxInputWords is the maximum number of words sent to the LLM.  The first
// ~1500 words are more than enough for summary generation and keep token
// usage reasonable for both cloud APIs and local models.
const maxInputWords = 1500

// maxSummaryLen is the hard upper limit (in runes) for AI-generated summaries.
// Summaries exceeding this are truncated at the nearest sentence or word
// boundary by truncateSentence.
const maxSummaryLen = 300

// reStructuredPatterns detects structured output that is NOT a valid
// single-paragraph summary.
var reStructuredPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^\d+\.\s`),                                          // numbered list: "1. Foo"
	regexp.MustCompile(`(?mi)^[A-Z]\.\s`),                                       // lettered list: "A. Foo"
	regexp.MustCompile(`(?m)^[-*+]\s`),                                          // bullet list
	regexp.MustCompile(`(?mi)^(resumen|summary|esquema|conclusion|contexto)\b`), // labels/titles
	regexp.MustCompile(`(?m)^.{0,40}:\s*$`),                                     // heading-like: "Key points:"
}

// isStructuredOutput returns true if the text contains patterns typical of
// structured LLM output (numbered lists, headings, labels) rather than a
// single paragraph summary.
func isStructuredOutput(text string) bool {
	matches := 0
	for _, re := range reStructuredPatterns {
		if re.MatchString(text) {
			matches++
		}
	}
	// A single match could be a false positive (e.g. "Resumen:" as first
	// word then a proper paragraph).  Require 2+ pattern matches or check
	// for strong indicators.
	if matches >= 2 {
		return true
	}
	// Also reject if the text has too many newlines (structured output).
	lines := strings.Count(text, "\n")
	return lines > 4
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
	// Kairos's llm.NewOllama hardcodes a 120s HTTP client timeout
	// that silently overrides cfg.AI.Timeout for slow local models.
	// newOllamaClient (internal/summary/ollama.go) returns an
	// http.Client without a built-in timeout so the request context
	// — which the caller already wraps with cfg.AI.Timeout — is the
	// only deadline that fires.
	return newOllamaClient(cfg.BaseURL), nil
}
