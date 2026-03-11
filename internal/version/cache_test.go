package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheRoundTrip(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	versions := &Versions{
		Armoctl:     "v1.0.0",
		Operator:    "v2.0.0",
		PtraceAgent: "v3.0.0",
	}

	// Save to cache
	err := SaveCache(versions)
	if err != nil {
		t.Fatalf("SaveCache() error = %v", err)
	}

	// Load from cache
	cached := LoadCache()
	if cached == nil {
		t.Fatal("LoadCache() returned nil")
	}

	// Verify versions
	if cached.Versions.Armoctl != versions.Armoctl {
		t.Errorf("Armoctl = %v, want %v", cached.Versions.Armoctl, versions.Armoctl)
	}
	if cached.Versions.Operator != versions.Operator {
		t.Errorf("Operator = %v, want %v", cached.Versions.Operator, versions.Operator)
	}
	if cached.Versions.PtraceAgent != versions.PtraceAgent {
		t.Errorf("PtraceAgent = %v, want %v", cached.Versions.PtraceAgent, versions.PtraceAgent)
	}

	// Verify timestamp is recent
	if time.Since(cached.FetchedAt) > time.Second {
		t.Errorf("FetchedAt is too old: %v", cached.FetchedAt)
	}
}

func TestLoadCache_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Should return nil when cache doesn't exist
	cached := LoadCache()
	if cached != nil {
		t.Errorf("LoadCache() = %v, want nil for non-existent cache", cached)
	}
}

func TestLoadCache_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create cache directory and invalid file
	cacheDir := filepath.Join(tmpDir, ".armoctl", "cache")
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(filepath.Join(cacheDir, "versions.json"), []byte("not json"), 0644)

	// Should return nil for invalid JSON
	cached := LoadCache()
	if cached != nil {
		t.Errorf("LoadCache() = %v, want nil for invalid JSON", cached)
	}
}

func TestIsCacheStale(t *testing.T) {
	tests := []struct {
		name      string
		cached    *CachedVersions
		wantStale bool
	}{
		{
			name:      "nil cache is stale",
			cached:    nil,
			wantStale: true,
		},
		{
			name: "fresh cache is not stale",
			cached: &CachedVersions{
				FetchedAt: time.Now(),
				Versions:  Versions{Armoctl: "v1.0.0"},
			},
			wantStale: false,
		},
		{
			name: "old cache is stale",
			cached: &CachedVersions{
				FetchedAt: time.Now().Add(-25 * time.Hour), // Older than CacheTTL
				Versions:  Versions{Armoctl: "v1.0.0"},
			},
			wantStale: true,
		},
		{
			name: "cache at TTL boundary is not stale",
			cached: &CachedVersions{
				FetchedAt: time.Now().Add(-23 * time.Hour), // Just under CacheTTL
				Versions:  Versions{Armoctl: "v1.0.0"},
			},
			wantStale: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCacheStale(tt.cached); got != tt.wantStale {
				t.Errorf("IsCacheStale() = %v, want %v", got, tt.wantStale)
			}
		})
	}
}

func TestCachePath(t *testing.T) {
	path, err := CachePath()
	if err != nil {
		t.Fatalf("CachePath() error = %v", err)
	}

	if path == "" {
		t.Error("CachePath() returned empty string")
	}

	// Should contain the expected path components
	if filepath.Base(path) != CacheFile {
		t.Errorf("CachePath() basename = %v, want %v", filepath.Base(path), CacheFile)
	}
}

func TestEnsureCacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	err := EnsureCacheDir()
	if err != nil {
		t.Fatalf("EnsureCacheDir() error = %v", err)
	}

	// Check directory was created
	cacheDir := filepath.Join(tmpDir, ".armoctl", "cache")
	info, err := os.Stat(cacheDir)
	if err != nil {
		t.Fatalf("Cache directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Cache path is not a directory")
	}
}

func TestSaveCache_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	versions := &Versions{Armoctl: "v1.0.0"}

	// Directory doesn't exist yet
	err := SaveCache(versions)
	if err != nil {
		t.Fatalf("SaveCache() error = %v", err)
	}

	// Verify file was created
	cachePath := filepath.Join(tmpDir, ".armoctl", "cache", "versions.json")
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Cache file was not created")
	}
}

func TestCacheFileFormat(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	versions := &Versions{
		Armoctl:     "v1.0.0",
		Operator:    "v2.0.0",
		PtraceAgent: "v3.0.0",
	}

	err := SaveCache(versions)
	if err != nil {
		t.Fatalf("SaveCache() error = %v", err)
	}

	// Read raw file and verify JSON structure
	cachePath := filepath.Join(tmpDir, ".armoctl", "cache", "versions.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to read cache file: %v", err)
	}

	var cached CachedVersions
	if err := json.Unmarshal(data, &cached); err != nil {
		t.Fatalf("Failed to parse cache file: %v", err)
	}

	if cached.Versions.Armoctl != "v1.0.0" {
		t.Errorf("Armoctl = %v, want v1.0.0", cached.Versions.Armoctl)
	}
}
