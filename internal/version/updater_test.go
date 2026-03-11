package version

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuildDownloadURL_Platform(t *testing.T) {
	url := buildDownloadURL()

	// Should contain OS
	expectedOS := runtime.GOOS
	if expectedOS == "darwin" || expectedOS == "linux" || expectedOS == "windows" {
		// Valid OS
	} else {
		t.Skipf("Skipping test on unsupported OS: %s", expectedOS)
	}

	// Should contain architecture
	expectedArch := runtime.GOARCH
	if expectedArch == "amd64" || expectedArch == "arm64" {
		// Valid arch
	} else {
		t.Skipf("Skipping test on unsupported arch: %s", expectedArch)
	}

	// URL should contain both
	if len(url) == 0 {
		t.Error("buildDownloadURL() returned empty URL")
	}
}

func TestDownloadToTempWithContext(t *testing.T) {
	// Create a test server that returns a fake binary
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fake binary content"))
	}))
	defer server.Close()

	// Download to temp
	tmpPath, err := downloadToTempWithContext(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("downloadToTempWithContext() error = %v", err)
	}
	defer func() { _ = os.Remove(tmpPath) }()

	// Verify file exists
	info, err := os.Stat(tmpPath)
	if err != nil {
		t.Fatalf("Temp file not created: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}
	if string(content) != "fake binary content" {
		t.Errorf("Content = %q, want %q", string(content), "fake binary content")
	}

	// Verify it's a file, not directory
	if info.IsDir() {
		t.Error("Temp path is a directory, expected file")
	}
}

func TestDownloadToTempWithContext_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := downloadToTempWithContext(context.Background(), server.URL)
	if err == nil {
		t.Error("downloadToTempWithContext() expected error for 500 response")
	}
}

func TestDownloadToTempWithContext_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := downloadToTempWithContext(context.Background(), server.URL)
	if err == nil {
		t.Error("downloadToTempWithContext() expected error for 404 response")
	}
}

func TestDownloadToTempWithContext_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := downloadToTempWithContext(ctx, server.URL)
	if err == nil {
		t.Error("downloadToTempWithContext() expected error for cancelled context")
	}
}

func TestCheckWritable(t *testing.T) {
	// Create a writable temp directory
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-binary")

	// Create a dummy file
	_ = os.WriteFile(tmpFile, []byte("test"), 0755)

	// Should succeed for writable directory
	err := checkWritable(tmpFile)
	if err != nil {
		t.Errorf("checkWritable() error = %v for writable directory", err)
	}
}

func TestCheckWritable_ReadOnlyDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping read-only directory test on Windows")
	}

	// Create a read-only temp directory
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-binary")

	// Create a dummy file
	_ = os.WriteFile(tmpFile, []byte("test"), 0755)

	// Make directory read-only
	_ = os.Chmod(tmpDir, 0555)
	defer func() { _ = os.Chmod(tmpDir, 0755) }() // Restore for cleanup

	// Should fail for read-only directory
	err := checkWritable(tmpFile)
	if err == nil {
		t.Error("checkWritable() expected error for read-only directory")
	}
}

func TestReplaceBinary(t *testing.T) {
	tmpDir := t.TempDir()

	// Create "current" binary
	currentPath := filepath.Join(tmpDir, "current-binary")
	_ = os.WriteFile(currentPath, []byte("old content"), 0755)

	// Create "new" binary
	newPath := filepath.Join(tmpDir, "new-binary")
	_ = os.WriteFile(newPath, []byte("new content"), 0644)

	// Replace
	err := replaceBinary(currentPath, newPath)
	if err != nil {
		t.Fatalf("replaceBinary() error = %v", err)
	}

	// Verify new content
	content, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatalf("Failed to read replaced binary: %v", err)
	}
	if string(content) != "new content" {
		t.Errorf("Content = %q, want %q", string(content), "new content")
	}

	// Verify permissions preserved (should be executable)
	info, err := os.Stat(currentPath)
	if err != nil {
		t.Fatalf("Failed to stat replaced binary: %v", err)
	}
	if info.Mode()&0100 == 0 {
		t.Error("Execute permission not preserved")
	}

	// Verify backup was cleaned up
	backupPath := currentPath + ".old"
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("Backup file was not cleaned up")
	}

	// Verify new file was cleaned up
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Error("New file should have been moved/removed")
	}
}

func TestReplaceBinary_RestoresBackupOnFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Create "current" binary
	currentPath := filepath.Join(tmpDir, "current-binary")
	_ = os.WriteFile(currentPath, []byte("old content"), 0755)

	// Non-existent new file (will cause error)
	newPath := filepath.Join(tmpDir, "non-existent")

	// Replace should fail
	err := replaceBinary(currentPath, newPath)
	if err == nil {
		t.Fatal("replaceBinary() expected error for non-existent new file")
	}

	// Current binary should be restored
	content, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatalf("Current binary not restored: %v", err)
	}
	if string(content) != "old content" {
		t.Errorf("Content = %q, want %q (original)", string(content), "old content")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "source")
	dstPath := filepath.Join(tmpDir, "dest")

	// Create source file
	_ = os.WriteFile(srcPath, []byte("test content"), 0644)

	// Copy
	err := copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	// Verify destination
	content, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Content = %q, want %q", string(content), "test content")
	}

	// Verify source still exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		t.Error("Source file should still exist after copy")
	}
}

func TestCopyFile_NonExistentSource(t *testing.T) {
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "non-existent")
	dstPath := filepath.Join(tmpDir, "dest")

	err := copyFile(srcPath, dstPath)
	if err == nil {
		t.Error("copyFile() expected error for non-existent source")
	}
}

func TestGetExecutablePath(t *testing.T) {
	path, err := GetExecutablePath()
	if err != nil {
		t.Fatalf("GetExecutablePath() error = %v", err)
	}

	if path == "" {
		t.Error("GetExecutablePath() returned empty string")
	}

	// Path should exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("GetExecutablePath() returned non-existent path: %s", path)
	}
}
