package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"osg/internal/config"
)

// AICacheEntry stores a single cached AI summary together with metadata
// about how and when it was generated.
type AICacheEntry struct {
	Summary     string `json:"summary"`
	Provider    string `json:"provider,omitempty"`
	Model       string `json:"model,omitempty"`
	GeneratedAt string `json:"generated_at"`
}

// AICache is the in-memory representation of the AI summary cache.
// It is safe for concurrent use from multiple goroutines.
type AICache struct {
	mu      sync.RWMutex
	Entries map[string]AICacheEntry `json:"entries"`

	// provider and model are set at construction and stamped into new entries.
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

// Len returns the number of cached entries.
func (c *AICache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Entries)
}

// ---------------------------------------------------------------------------
// Persistence
// ---------------------------------------------------------------------------

const aiCacheFile = "ai-summaries.json"

// aiCachePath returns the full path to the AI summary cache file inside
// the configured build cache directory.
func aiCachePath(cfg config.Config) string {
	dir := cfg.BuildCacheDir
	if dir == "" {
		dir = ".osg/cache"
	}
	return filepath.Join(dir, aiCacheFile)
}

// loadAICache reads the cache file from disk.  If the file doesn't exist
// or is unreadable an empty cache is returned (never nil).
func loadAICache(path string, logger *slog.Logger) *AICache {
	cache := newAICache("", "")

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) && logger != nil {
			logger.Warn("failed to read AI summary cache", "path", path, "error", err)
		}
		return cache
	}

	if err := json.Unmarshal(data, cache); err != nil {
		if logger != nil {
			logger.Warn("failed to parse AI summary cache, starting fresh", "path", path, "error", err)
		}
		return newAICache("", "")
	}

	if cache.Entries == nil {
		cache.Entries = make(map[string]AICacheEntry)
	}

	if logger != nil {
		logger.Info("AI summary cache loaded", "entries", len(cache.Entries))
	}
	return cache
}

// saveAICache writes the cache to disk, creating parent directories as
// needed.  Returns any write error.
func saveAICache(path string, cache *AICache, logger *slog.Logger) error {
	if cache == nil {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	cache.mu.RLock()
	data, err := json.MarshalIndent(cache, "", "  ")
	cache.mu.RUnlock()
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}

	if logger != nil {
		logger.Info("AI summary cache saved", "entries", cache.Len(), "path", path)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Hashing
// ---------------------------------------------------------------------------

// contentHash returns the SHA-256 hex digest of the given content string.
// This is used as the cache key — when content changes, the hash changes
// and the old cache entry is naturally invalidated.
func contentHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return hex.EncodeToString(h[:])
}
