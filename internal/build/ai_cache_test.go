package build

import (
	"sync"
	"testing"

	"osg/internal/config"
)

// ---------------------------------------------------------------------------
// contentHash
// ---------------------------------------------------------------------------

func TestContentHash_Deterministic(t *testing.T) {
	a := contentHash("hello world")
	b := contentHash("hello world")
	if a != b {
		t.Errorf("expected same hash, got %q and %q", a, b)
	}
}

func TestContentHash_DifferentContent(t *testing.T) {
	a := contentHash("hello")
	b := contentHash("world")
	if a == b {
		t.Error("expected different hashes for different content")
	}
}

func TestContentHash_Length(t *testing.T) {
	h := contentHash("test")
	// SHA-256 produces 64 hex characters.
	if len(h) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(h))
	}
}

// ---------------------------------------------------------------------------
// AICache in-memory operations
// ---------------------------------------------------------------------------

func TestAICache_StoreAndLookup(t *testing.T) {
	cache := newAICache("gemini", "gemini-3-flash-preview")
	cache.Store("abc123", "Test summary.")

	entry, ok := cache.Lookup("abc123")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if entry.Summary != "Test summary." {
		t.Errorf("expected %q, got %q", "Test summary.", entry.Summary)
	}
	if entry.Provider != "gemini" {
		t.Errorf("expected provider %q, got %q", "gemini", entry.Provider)
	}
	if entry.Model != "gemini-3-flash-preview" {
		t.Errorf("expected model %q, got %q", "gemini-3-flash-preview", entry.Model)
	}
	if entry.GeneratedAt == "" {
		t.Error("expected non-empty GeneratedAt")
	}
}

func TestAICache_LookupMiss(t *testing.T) {
	cache := newAICache("", "")
	_, ok := cache.Lookup("nonexistent")
	if ok {
		t.Error("expected cache miss")
	}
}

func TestAICache_Overwrite(t *testing.T) {
	cache := newAICache("gemini", "model")
	cache.Store("key1", "first")
	cache.Store("key1", "second")

	entry, ok := cache.Lookup("key1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if entry.Summary != "second" {
		t.Errorf("expected overwritten value %q, got %q", "second", entry.Summary)
	}
}

func TestAICache_Len(t *testing.T) {
	cache := newAICache("", "")
	if cache.Len() != 0 {
		t.Errorf("expected 0, got %d", cache.Len())
	}
	cache.Store("a", "sa")
	cache.Store("b", "sb")
	if cache.Len() != 2 {
		t.Errorf("expected 2, got %d", cache.Len())
	}
}

func TestAICache_ConcurrentAccess(t *testing.T) {
	cache := newAICache("test", "model")
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n * 2) // n writers + n readers

	for i := range n {
		go func(idx int) {
			defer wg.Done()
			hash := contentHash("content" + string(rune('0'+idx%10)))
			cache.Store(hash, "summary")
		}(i)
		go func(idx int) {
			defer wg.Done()
			hash := contentHash("content" + string(rune('0'+idx%10)))
			cache.Lookup(hash)
		}(i)
	}

	wg.Wait()
	// No panic = pass (testing for data races).
}

// ---------------------------------------------------------------------------
// SQLite-backed save/load round-trip
// ---------------------------------------------------------------------------

func cacheTestCfg(t *testing.T) config.Config {
	t.Helper()
	return config.Config{BuildCacheDir: t.TempDir()}
}

func TestAICache_SaveLoadRoundTrip(t *testing.T) {
	cfg := cacheTestCfg(t)

	cache := newAICache("anthropic", "claude")
	cache.Store(contentHash("post one"), "Summary one.")
	cache.Store(contentHash("post two"), "Summary two.")

	if err := saveAICache(cfg, cache, nil); err != nil {
		t.Fatalf("saveAICache: %v", err)
	}

	loaded := loadAICache(cfg, nil)
	if loaded.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", loaded.Len())
	}

	entry, ok := loaded.Lookup(contentHash("post one"))
	if !ok {
		t.Fatal("expected cache hit for 'post one'")
	}
	if entry.Summary != "Summary one." {
		t.Errorf("expected %q, got %q", "Summary one.", entry.Summary)
	}
	if entry.Provider != "anthropic" {
		t.Errorf("expected provider %q, got %q", "anthropic", entry.Provider)
	}
}

func TestAICache_LoadEmptyStore(t *testing.T) {
	cfg := cacheTestCfg(t)
	cache := loadAICache(cfg, nil)
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	if cache.Len() != 0 {
		t.Errorf("expected empty cache, got %d entries", cache.Len())
	}
}

func TestAICache_SaveCreatesDirectories(t *testing.T) {
	cfg := config.Config{BuildCacheDir: t.TempDir() + "/a/b/c"}

	cache := newAICache("", "")
	cache.Store("hash", "summary")

	if err := saveAICache(cfg, cache, nil); err != nil {
		t.Fatalf("saveAICache: %v", err)
	}

	loaded := loadAICache(cfg, nil)
	if _, ok := loaded.Lookup("hash"); !ok {
		t.Error("expected entry to be persisted under nested dir")
	}
}

func TestAICache_SaveNilCache(t *testing.T) {
	cfg := cacheTestCfg(t)
	if err := saveAICache(cfg, nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Cache invalidation via content hash
// ---------------------------------------------------------------------------

func TestAICache_ContentChangeInvalidates(t *testing.T) {
	cache := newAICache("", "")

	original := "Original blog content."
	hash1 := contentHash(original)
	cache.Store(hash1, "Original summary.")

	// Content changes -> different hash -> cache miss.
	modified := "Modified blog content with new paragraph."
	hash2 := contentHash(modified)

	_, ok := cache.Lookup(hash2)
	if ok {
		t.Error("expected cache miss for modified content")
	}

	// Original still in cache.
	entry, ok := cache.Lookup(hash1)
	if !ok {
		t.Error("expected original still cached")
	}
	if entry.Summary != "Original summary." {
		t.Errorf("got %q", entry.Summary)
	}
}

func TestAICache_Remove(t *testing.T) {
	cache := newAICache("provider", "model")
	cache.Store("a", "summary a")
	cache.Store("b", "summary b")

	if !cache.Remove("a") {
		t.Error("Remove(a) returned false; expected true on existing key")
	}
	if _, ok := cache.Lookup("a"); ok {
		t.Error("expected entry a to be gone after Remove")
	}
	if _, ok := cache.Lookup("b"); !ok {
		t.Error("entry b should still be present")
	}
	if cache.Remove("a") {
		t.Error("Remove(a) returned true on second call; expected false")
	}
}

// ---------------------------------------------------------------------------
// SQLite store invalidation API
// ---------------------------------------------------------------------------

func TestInvalidateAISummary_RemovesPersistedEntry(t *testing.T) {
	cfg := cacheTestCfg(t)
	cache := newAICache("p", "m")
	cache.Store("zzz", "to be invalidated")
	if err := saveAICache(cfg, cache, nil); err != nil {
		t.Fatalf("saveAICache: %v", err)
	}

	removed, err := InvalidateAISummary(cfg, "zzz", nil)
	if err != nil {
		t.Fatalf("InvalidateAISummary: %v", err)
	}
	if !removed {
		t.Error("expected removed=true for existing entry")
	}

	// Round-trip: loading again should not surface the entry.
	loaded := loadAICache(cfg, nil)
	if _, ok := loaded.Lookup("zzz"); ok {
		t.Error("expected entry to be gone after invalidation")
	}

	// Second invalidation is a no-op.
	removed, err = InvalidateAISummary(cfg, "zzz", nil)
	if err != nil {
		t.Fatalf("InvalidateAISummary (second): %v", err)
	}
	if removed {
		t.Error("expected removed=false for missing entry")
	}
}

func TestLookupAISummary_ReturnsCachedValue(t *testing.T) {
	cfg := cacheTestCfg(t)
	cache := newAICache("p", "m")
	cache.Store("hh", "cached value")
	if err := saveAICache(cfg, cache, nil); err != nil {
		t.Fatalf("saveAICache: %v", err)
	}

	got, ok := LookupAISummary(cfg, "hh", nil)
	if !ok {
		t.Fatal("expected lookup hit")
	}
	if got != "cached value" {
		t.Errorf("got %q, want %q", got, "cached value")
	}

	if _, ok := LookupAISummary(cfg, "missing", nil); ok {
		t.Error("expected miss for unknown hash")
	}
}
