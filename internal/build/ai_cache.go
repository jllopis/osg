package build

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"sync"
	"time"

	"osg/internal/config"
)

// AICacheEntry is a single AI-generated summary plus the metadata
// recorded the moment it was produced. Persisted in
// .osg/cache/summaries.db (see summary_store.go).
type AICacheEntry struct {
	Summary     string
	Provider    string
	Model       string
	GeneratedAt string
}

// AICache is the in-memory view of every cached AI summary. It is
// loaded from SQLite at the start of a build, mutated by the parallel
// summary pipeline, and flushed back at the end. Safe for concurrent
// access.
type AICache struct {
	mu      sync.RWMutex
	Entries map[string]AICacheEntry

	// provider and model are stamped onto new entries so the row
	// records which backend produced the value.
	provider string
	model    string
}

// newAICache creates an empty cache tagged with the given provider/model.
func newAICache(provider, model string) *AICache {
	return &AICache{
		Entries:  make(map[string]AICacheEntry),
		provider: provider,
		model:    model,
	}
}

// Lookup returns the cached entry for the given content hash, if any.
func (c *AICache) Lookup(hash string) (AICacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.Entries[hash]
	return entry, ok
}

// Store adds or replaces a cache entry keyed by content hash.
func (c *AICache) Store(hash string, summary string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Entries[hash] = AICacheEntry{
		Summary:     summary,
		Provider:    c.provider,
		Model:       c.model,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// Remove drops the entry with the given content hash. Returns true
// when an entry was actually removed.
func (c *AICache) Remove(hash string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.Entries[hash]; !ok {
		return false
	}
	delete(c.Entries, hash)
	return true
}

// Len returns the number of cached entries.
func (c *AICache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Entries)
}

// loadAICache loads every persisted summary into a fresh in-memory
// cache. Backed by SQLite (.osg/cache/summaries.db).
func loadAICache(cfg config.Config, logger *slog.Logger) *AICache {
	return loadAISummariesIntoCache(cfg, logger)
}

// saveAICache persists the in-memory cache to SQLite.
func saveAICache(cfg config.Config, cache *AICache, logger *slog.Logger) error {
	return persistAICache(cfg, cache, logger)
}

// InvalidateAISummary drops the entry with the given content hash
// from the persisted store so the next build is forced to re-call
// the LLM for that page. Returns true when an entry was removed.
func InvalidateAISummary(cfg config.Config, hash string, logger *slog.Logger) (bool, error) {
	store, err := openSummaryStore(cfg)
	if err != nil {
		return false, err
	}
	defer func() { _ = store.Close() }()
	removed, err := store.Remove(hash)
	if err != nil {
		return false, err
	}
	if logger != nil && removed {
		logger.Info("ai summary invalidated", "hash", hash)
	}
	return removed, nil
}

// LoadAISummaries returns a snapshot of every cached entry as a map
// hash → summary. Used by the UI when rendering a page list so it can
// look up cached values without opening the SQLite handle once per
// page. Logs and returns an empty map on any failure (keeping the
// page list responsive even if the store is missing or corrupt).
func LoadAISummaries(cfg config.Config, logger *slog.Logger) map[string]string {
	store, err := openSummaryStore(cfg)
	if err != nil {
		if logger != nil {
			logger.Warn("open summary store failed", "error", err)
		}
		return map[string]string{}
	}
	defer func() { _ = store.Close() }()
	rows, err := store.LoadAll()
	if err != nil {
		if logger != nil {
			logger.Warn("load summaries failed", "error", err)
		}
		return map[string]string{}
	}
	out := make(map[string]string, len(rows))
	for h, e := range rows {
		out[h] = e.Summary
	}
	return out
}

// UpsertAISummary writes (or replaces) a single cached entry. Used
// by the UI's "regenerate now" action to persist a freshly generated
// summary without going through the build pipeline.
func UpsertAISummary(cfg config.Config, hash, summary, provider, model string, logger *slog.Logger) error {
	store, err := openSummaryStore(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()
	if err := store.Upsert(hash, AICacheEntry{
		Summary:     summary,
		Provider:    provider,
		Model:       model,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}
	if logger != nil {
		logger.Info("ai summary upserted", "hash", hash, "provider", provider, "model", model)
	}
	return nil
}

// LookupAISummary returns the cached AI summary for the given content
// hash, or "" with ok=false when no entry exists. Read-only — used by
// the UI to preview the cached value next to the editor.
func LookupAISummary(cfg config.Config, hash string, logger *slog.Logger) (string, bool) {
	store, err := openSummaryStore(cfg)
	if err != nil {
		if logger != nil {
			logger.Warn("open summary store failed", "error", err)
		}
		return "", false
	}
	defer func() { _ = store.Close() }()
	entry, ok, err := store.Lookup(hash)
	if err != nil {
		if logger != nil {
			logger.Warn("lookup summary failed", "error", err)
		}
		return "", false
	}
	if !ok {
		return "", false
	}
	return entry.Summary, true
}

// contentHash returns the SHA-256 hex digest of the given content
// string. This is the cache key — when content changes the hash
// changes and the old entry naturally goes unused.
func contentHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}

// ContentHash exposes contentHash so callers in other packages can
// compute the same cache key the build uses.
func ContentHash(content string) string {
	return contentHash(content)
}
