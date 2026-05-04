package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// CacheTTL is how long cached version info is considered valid.
	CacheTTL = 1 * time.Hour

	// CacheDir is the subdirectory under ~/.armoctl for cache files.
	CacheDir = "cache"

	// CacheFile is the name of the legacy multi-agent versions cache.
	// Read by ECS image helpers; never written anymore (the fetch path
	// that used to populate it has been removed in favour of the
	// per-component CDN /latest.txt published by each release).
	CacheFile = "versions.json"
)

// CachedVersions holds the on-disk multi-agent version cache shape.
// Only ECS image lookup reads this; nothing currently populates it.
type CachedVersions struct {
	FetchedAt time.Time `json:"fetched_at"`
	Versions  Versions  `json:"versions"`
}

// CachePath returns the full path to the multi-agent cache file.
func CachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".armoctl", CacheDir, CacheFile), nil
}

// EnsureCacheDir creates the cache directory if it doesn't exist.
func EnsureCacheDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cacheDir := filepath.Join(home, ".armoctl", CacheDir)
	return os.MkdirAll(cacheDir, 0755)
}

// LoadCache reads the cached multi-agent version info from disk.
// Returns nil if the cache doesn't exist or is invalid — ECS callers
// then fall back to FallbackTag.
func LoadCache() *CachedVersions {
	path, err := CachePath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cached CachedVersions
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil
	}
	return &cached
}
