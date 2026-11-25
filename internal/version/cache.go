package version

import (
	"os"
	"path/filepath"
	"time"
)

const (
	// CacheTTL is how long cached version info is considered valid.
	CacheTTL = 1 * time.Hour

	// CacheDir is the subdirectory under ~/.armoctl for cache files.
	CacheDir = "cache"
)

// EnsureCacheDir creates the cache directory if it doesn't exist.
func EnsureCacheDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cacheDir := filepath.Join(home, ".armoctl", CacheDir)
	return os.MkdirAll(cacheDir, 0755)
}
