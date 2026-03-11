package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// CacheTTL is how long the cached version info is considered valid.
	CacheTTL = 1 * time.Hour

	// CacheDir is the subdirectory under ~/.armoctl for cache files.
	CacheDir = "cache"

	// CacheFile is the name of the cached versions file.
	CacheFile = "versions.json"
)

// CachedVersions holds version info along with fetch timestamp.
type CachedVersions struct {
	FetchedAt time.Time `json:"fetched_at"`
	Versions  Versions  `json:"versions"`
}

// CachePath returns the full path to the cache file.
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

// LoadCache reads the cached version info from disk.
// Returns nil if cache doesn't exist or is invalid.
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

// SaveCache writes version info to the cache file.
func SaveCache(versions *Versions) error {
	if err := EnsureCacheDir(); err != nil {
		return err
	}

	path, err := CachePath()
	if err != nil {
		return err
	}

	cached := CachedVersions{
		FetchedAt: time.Now(),
		Versions:  *versions,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// IsCacheStale returns true if the cache is older than CacheTTL or doesn't exist.
func IsCacheStale(cached *CachedVersions) bool {
	if cached == nil {
		return true
	}
	return time.Since(cached.FetchedAt) > CacheTTL
}

// GetLatestVersions returns the latest versions, using cache if valid,
// otherwise fetching from remote.
func GetLatestVersions() (*Versions, error) {
	cached := LoadCache()

	if !IsCacheStale(cached) {
		return &cached.Versions, nil
	}

	// Cache is stale or missing, fetch fresh
	versions, err := FetchLatest()
	if err != nil {
		// If fetch fails but we have stale cache, use it
		if cached != nil {
			return &cached.Versions, nil
		}
		return nil, err
	}

	// Save to cache (ignore errors - caching is best effort)
	_ = SaveCache(versions)

	return versions, nil
}
