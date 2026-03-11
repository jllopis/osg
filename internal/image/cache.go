package image

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

const imageCachePath = ".osg/cache/images.json"

// imageCache persists optimization results keyed by source file hash.
type imageCache struct {
	Version int                    `json:"version"`
	Entries map[string]*cacheEntry `json:"entries"`
}

// cacheEntry stores the result for a single source image.
type cacheEntry struct {
	Hash   string  `json:"hash"`   // SHA-256 of source file content
	Result *Result `json:"result"` // optimization result (variants, widths, etc.)
	Count  int     `json:"count"`  // number of variants generated
}

const imageCacheVersion = 1

func loadImageCache() *imageCache {
	data, err := os.ReadFile(imageCachePath)
	if err != nil {
		return &imageCache{Version: imageCacheVersion, Entries: map[string]*cacheEntry{}}
	}
	var c imageCache
	if err := json.Unmarshal(data, &c); err != nil || c.Version != imageCacheVersion {
		return &imageCache{Version: imageCacheVersion, Entries: map[string]*cacheEntry{}}
	}
	if c.Entries == nil {
		c.Entries = map[string]*cacheEntry{}
	}
	return &c
}

func saveImageCache(c *imageCache) error {
	dir := filepath.Dir(imageCachePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(imageCachePath, data, 0o644)
}

// hashFile returns the SHA-256 hex digest of a file's contents.
func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// variantsExist checks that all variant files referenced by a Result
// actually exist on disk in the given publicDir.
func variantsExist(publicDir string, res *Result) bool {
	if res == nil {
		return false
	}
	for _, variants := range res.Variants {
		for _, v := range variants {
			// v.URLPath is like "/img/hero-640w.webp", translate to filesystem.
			fp := filepath.Join(publicDir, filepath.FromSlash(v.URLPath))
			if _, err := os.Stat(fp); err != nil {
				return false
			}
		}
	}
	return true
}
