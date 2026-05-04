package version

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// The multi-agent fetch path that used to populate this cache has been
// removed; the cache file is read-only now (consumed by ECS image
// helpers, populated by nothing). These tests cover the read+layout
// invariants that ECS still depends on.

func TestLoadCache_RoundTrip_ReadOnly(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Hand-write a cache file the way ECS would have observed one
	// historically and verify LoadCache parses it.
	dir := filepath.Join(tmpDir, ".armoctl", "cache")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	want := CachedVersions{
		FetchedAt: time.Now(),
		Versions: Versions{
			Armoctl:     "v1.0.0",
			ECSOperator: "v2.0.0",
			PtraceAgent: "v3.0.0",
		},
	}
	body, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, CacheFile), body, 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadCache()
	if got == nil {
		t.Fatal("LoadCache() returned nil for a valid cache file")
	}
	if got.Versions.PtraceAgent != "v3.0.0" || got.Versions.ECSOperator != "v2.0.0" {
		t.Errorf("LoadCache() = %+v, want PtraceAgent=v3.0.0 ECSOperator=v2.0.0", got.Versions)
	}
}

func TestLoadCache_NonExistent(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if got := LoadCache(); got != nil {
		t.Errorf("LoadCache() = %v, want nil for non-existent cache", got)
	}
}

func TestLoadCache_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cacheDir := filepath.Join(tmpDir, ".armoctl", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, CacheFile), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	if got := LoadCache(); got != nil {
		t.Errorf("LoadCache() = %v, want nil for invalid JSON", got)
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
	if filepath.Base(path) != CacheFile {
		t.Errorf("CachePath() basename = %v, want %v", filepath.Base(path), CacheFile)
	}
}

func TestEnsureCacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	if err := EnsureCacheDir(); err != nil {
		t.Fatalf("EnsureCacheDir() error = %v", err)
	}

	cacheDir := filepath.Join(tmpDir, ".armoctl", "cache")
	info, err := os.Stat(cacheDir)
	if err != nil {
		t.Fatalf("cache directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("cache path is not a directory")
	}
}
