package version

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	// DownloadTimeout is the timeout for downloading the binary.
	DownloadTimeout = 2 * time.Minute

	// MaxBinarySize is the maximum size of the binary download (100MB).
	MaxBinarySize = 100 * 1024 * 1024
)

// SelfUpdate downloads and replaces the current binary with the latest version.
func SelfUpdate() error {
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}

	// Resolve symlinks to get the actual binary location
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving symlinks: %w", err)
	}

	// Check if we can write to the binary location
	if err := checkWritable(execPath); err != nil {
		return err
	}

	// Build download URL
	url := buildDownloadURL()

	// Download to temp file
	tmpFile, err := downloadToTemp(url)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmpFile) }() // Clean up temp file on any error

	// Replace the binary
	if err := replaceBinary(execPath, tmpFile); err != nil {
		return err
	}

	return nil
}

// buildDownloadURL constructs the URL for the latest binary.
func buildDownloadURL() string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("%s/armoctl_latest_%s_%s%s",
		DistributionURL, runtime.GOOS, runtime.GOARCH, ext)
}

// checkWritable verifies we can write to the binary location.
func checkWritable(path string) error {
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("checking directory: %w", err)
	}

	_ = info // Used for the stat check

	// Try to create a temp file in the directory to check write access
	f, err := os.CreateTemp(dir, ".armoctl-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s: %w\nTry moving armoctl to a user-writable location", dir, err)
	}
	tmpName := f.Name()
	_ = f.Close()
	_ = os.Remove(tmpName)
	return nil
}

// downloadToTemp downloads the binary to a temporary file.
func downloadToTemp(url string) (string, error) {
	return downloadToTempWithContext(context.Background(), url)
}

// downloadToTempWithContext downloads the binary to a temporary file with context support.
func downloadToTempWithContext(ctx context.Context, url string) (string, error) {
	client := &http.Client{Timeout: DownloadTimeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading update: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create temp file
	tmpFile, err := os.CreateTemp("", "armoctl-update-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Limit download size to prevent OOM from malicious servers
	limitedReader := io.LimitReader(resp.Body, MaxBinarySize+1)

	// Copy download to temp file
	n, err := io.Copy(tmpFile, limitedReader)
	_ = tmpFile.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("writing update: %w", err)
	}
	if n > MaxBinarySize {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("download exceeds maximum size (%d bytes)", MaxBinarySize)
	}

	return tmpPath, nil
}

// replaceBinary replaces the current binary with the new one.
func replaceBinary(currentPath, newPath string) error {
	// Get current binary permissions
	info, err := os.Stat(currentPath)
	if err != nil {
		return fmt.Errorf("getting binary info: %w", err)
	}
	mode := info.Mode()

	// On Unix, we can replace a running binary by:
	// 1. Rename current to .old (atomic)
	// 2. Rename new to current (atomic)
	// 3. Remove .old
	//
	// On Windows, we can rename a running executable but cannot delete it.
	// The .old file will remain until the process exits or next cleanup.

	backupPath := currentPath + ".old"

	// Remove any existing backup (may fail on Windows if previous process still running)
	_ = os.Remove(backupPath)

	// Backup current binary
	if err := os.Rename(currentPath, backupPath); err != nil {
		return fmt.Errorf("backing up current binary: %w", err)
	}

	// Try to move new binary into place
	if err := os.Rename(newPath, currentPath); err != nil {
		// Rename might fail across filesystems, try copy instead
		if copyErr := copyFile(newPath, currentPath); copyErr != nil {
			// Try to restore backup
			_ = os.Rename(backupPath, currentPath)
			return fmt.Errorf("installing new binary: %w (copy also failed: %v)", err, copyErr)
		}
		// Copy succeeded, remove the temp file
		_ = os.Remove(newPath)
	}

	// Set correct permissions
	if err := os.Chmod(currentPath, mode); err != nil {
		// Non-fatal, try to continue
		fmt.Fprintf(os.Stderr, "Warning: could not set permissions: %v\n", err)
	}

	// Remove backup (may fail on Windows, which is fine)
	_ = os.Remove(backupPath)

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// GetExecutablePath returns the path to the current executable.
func GetExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(execPath)
}
