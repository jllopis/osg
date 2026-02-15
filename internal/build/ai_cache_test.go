package build

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
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
// Persistence: save / load round-trip
// ---------------------------------------------------------------------------

func TestAICache_SaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ai-summaries.json")

	// Create and populate a cache.
	cache := newAICache("anthropic", "claude")
	cache.Store(contentHash("post one"), "Summary one.")
	cache.Store(contentHash("post two"), "Summary two.")

	// Save.
	if err := saveAICache(path, cache, nil); err != nil {
		t.Fatalf("saveAICache: %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file not found: %v", err)
	}

	// Load into a new cache.
	loaded := loadAICache(path, nil)
	if loaded.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", loaded.Len())
	}

	hash1 := contentHash("post one")
	entry, ok := loaded.Lookup(hash1)
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

func TestAICache_LoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")

	cache := loadAICache(path, nil)
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	if cache.Len() != 0 {
		t.Errorf("expected empty cache, got %d entries", cache.Len())
	}
}

func TestAICache_LoadCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ai-summaries.json")

	// Write garbage.
	if err := os.WriteFile(path, []byte("not json{{{"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := loadAICache(path, nil)
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	if cache.Len() != 0 {
		t.Errorf("expected empty cache after corruption, got %d entries", cache.Len())
	}
}

func TestAICache_SaveCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "ai-summaries.json")

	cache := newAICache("", "")
	cache.Store("hash", "summary")

	if err := saveAICache(path, cache, nil); err != nil {
		t.Fatalf("saveAICache: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file not found: %v", err)
	}
}

func TestAICache_SaveNilCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ai-summaries.json")

	// Should not panic or error.
	if err := saveAICache(path, nil, nil); err != nil {
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
