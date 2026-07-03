package build

import (
	"strings"
	"testing"
	"time"

	"osg/internal/config"
	"osg/internal/site"
	"osg/internal/taxonomy"
)

// Preview builds (include_drafts) render scheduled pages so authors can
// check them, but feeds and the sitemap are public artifacts: neither may
// reveal draft or scheduled-future URLs.
func TestScheduledPagesExcludedFromFeedAndSitemap(t *testing.T) {
	published := &site.Page{
		Title: "Published", Path: "/published/", Permalink: "https://example.com/published/",
		Date: time.Now().Add(-24 * time.Hour),
	}
	scheduled := &site.Page{
		Title: "Scheduled", Path: "/scheduled/", Permalink: "https://example.com/scheduled/",
		Date:      time.Now(),
		PublishAt: time.Now().Add(24 * time.Hour),
	}
	draft := &site.Page{
		Title: "Draft", Path: "/draft/", Permalink: "https://example.com/draft/",
		Date: time.Now(), Draft: true,
	}

	views := feedPages([]*site.Page{published, scheduled, draft})
	if len(views) != 1 {
		t.Fatalf("feedPages returned %d entries, want 1", len(views))
	}
	if views[0]["title"] != "Published" {
		t.Errorf("feedPages kept %v, want the published page", views[0]["title"])
	}

	idx := site.New()
	idx.AddPage(published)
	idx.AddPage(scheduled)
	idx.AddPage(draft)
	cfg := config.Config{BaseURL: "https://example.com"}
	entries := collectSitemapEntries(cfg, idx, map[string]*taxonomy.Index{})
	for _, e := range entries {
		if strings.Contains(e.Permalink, "/scheduled/") || strings.Contains(e.Permalink, "/draft/") {
			t.Errorf("sitemap leaks unpublished URL %q", e.Permalink)
		}
	}
}
