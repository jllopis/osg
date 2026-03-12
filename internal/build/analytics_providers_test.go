package build

import (
	"strings"
	"testing"

	"osg/internal/config"
)

func TestAnalyticsSnippets_Empty(t *testing.T) {
	got := analyticsSnippets(nil)
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestAnalyticsSnippets_Cloudflare(t *testing.T) {
	providers := []config.AnalyticsProviderConfig{
		{Provider: "cloudflare", Token: "abc123"},
	}
	got := analyticsSnippets(providers)
	if !strings.Contains(got, `data-cf-beacon='{"token":"abc123"}'`) {
		t.Fatalf("missing Cloudflare beacon token:\n%s", got)
	}
	if !strings.Contains(got, "cloudflareinsights.com/beacon.min.js") {
		t.Fatalf("missing Cloudflare script URL:\n%s", got)
	}
}

func TestAnalyticsSnippets_Google(t *testing.T) {
	providers := []config.AnalyticsProviderConfig{
		{Provider: "google", TrackingID: "G-ABCDEF"},
	}
	got := analyticsSnippets(providers)
	if !strings.Contains(got, "googletagmanager.com/gtag/js?id=G-ABCDEF") {
		t.Fatalf("missing Google gtag script:\n%s", got)
	}
	if !strings.Contains(got, "gtag('config','G-ABCDEF')") {
		t.Fatalf("missing Google gtag config:\n%s", got)
	}
}

func TestAnalyticsSnippets_Plausible(t *testing.T) {
	providers := []config.AnalyticsProviderConfig{
		{Provider: "plausible", Domain: "example.com"},
	}
	got := analyticsSnippets(providers)
	if !strings.Contains(got, `data-domain="example.com"`) {
		t.Fatalf("missing Plausible domain:\n%s", got)
	}
	if !strings.Contains(got, "plausible.io/js/script.js") {
		t.Fatalf("missing Plausible script URL:\n%s", got)
	}
}

func TestAnalyticsSnippets_Fathom(t *testing.T) {
	providers := []config.AnalyticsProviderConfig{
		{Provider: "fathom", Token: "SITE123"},
	}
	got := analyticsSnippets(providers)
	if !strings.Contains(got, `data-site="SITE123"`) {
		t.Fatalf("missing Fathom site token:\n%s", got)
	}
	if !strings.Contains(got, "cdn.usefathom.com/script.js") {
		t.Fatalf("missing Fathom script URL:\n%s", got)
	}
}

func TestAnalyticsSnippets_Multiple(t *testing.T) {
	providers := []config.AnalyticsProviderConfig{
		{Provider: "cloudflare", Token: "cf-tok"},
		{Provider: "google", TrackingID: "G-123"},
	}
	got := analyticsSnippets(providers)
	if !strings.Contains(got, "cloudflareinsights") {
		t.Fatal("missing Cloudflare snippet")
	}
	if !strings.Contains(got, "googletagmanager") {
		t.Fatal("missing Google snippet")
	}
}

func TestAnalyticsSnippets_HTMLEscape(t *testing.T) {
	providers := []config.AnalyticsProviderConfig{
		{Provider: "cloudflare", Token: `<script>alert("xss")</script>`},
	}
	got := analyticsSnippets(providers)
	if strings.Contains(got, `<script>alert`) {
		t.Fatalf("XSS not escaped in token:\n%s", got)
	}
}
