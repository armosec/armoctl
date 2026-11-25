package version

import (
	"os"
	"path/filepath"
	"testing"
)

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
