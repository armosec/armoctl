package version

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ArmoctlLatestURL is the canonical CDN location of the latest armoctl
// version string, published by the release workflow alongside the
// binaries themselves. The file is plain text containing a single
// version tag (e.g. "v0.0.10\n").
//
// This used to be served via the cadashboardbe /api/v1/sensors/version
// endpoint, which sourced its value from a Helm value in the
// kubernetes-deployment repo. That field was bumped manually and went
// stale every release. Reading from the CDN — the same place
// install.sh already publishes the binaries — keeps the version-check
// honest with no platform-side coordination.
const (
	ArmoctlLatestURL  = DistributionURL + "/latest.txt"
	armoctlCacheFile  = "armoctl-latest.json"
	maxLatestTxtBytes = 64
)

// ArmoctlVersionCache holds the cached latest-armoctl version with a
// timestamp. Re-uses the same TTL semantics as the multi-agent cache
// (see CacheTTL).
type ArmoctlVersionCache struct {
	FetchedAt time.Time `json:"fetched_at"`
	Version   string    `json:"version"`
}

// armoctlCachePath returns the on-disk path for the cached armoctl
// version. Lives next to the multi-agent cache under ~/.armoctl/cache.
func armoctlCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".armoctl", CacheDir, armoctlCacheFile), nil
}

// FetchLatestArmoctl reads the latest armoctl version string from the
// release CDN. Public, unauthenticated, no customer-guid required.
func FetchLatestArmoctl(ctx context.Context) (string, error) {
	return FetchLatestArmoctlFrom(ctx, ArmoctlLatestURL)
}

// FetchLatestArmoctlFrom is the testable variant taking an explicit
// URL so tests can point at httptest.Server.URL.
func FetchLatestArmoctlFrom(ctx context.Context, url string) (string, error) {
	client := &http.Client{Timeout: FetchTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching latest armoctl version: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLatestTxtBytes))
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	v := strings.TrimSpace(string(body))
	// Reject HTML responses (CDN error pages, SPA fall-throughs) so a
	// misconfigured edge doesn't get parsed as a version string.
	if v == "" || strings.ContainsAny(v, "<>") {
		return "", fmt.Errorf("unexpected response body: %q", v)
	}
	return v, nil
}

// loadArmoctlCache reads and unmarshals the on-disk cache. Returns
// nil on any error so callers can simply re-fetch.
func loadArmoctlCache() *ArmoctlVersionCache {
	path, err := armoctlCachePath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var c ArmoctlVersionCache
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	return &c
}

// saveArmoctlCache writes the supplied version with the current
// timestamp. Best-effort: any error is returned but callers are
// expected to ignore it (caching is opportunistic).
func saveArmoctlCache(version string) error {
	if err := EnsureCacheDir(); err != nil {
		return err
	}
	path, err := armoctlCachePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(ArmoctlVersionCache{
		FetchedAt: time.Now(),
		Version:   version,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GetLatestArmoctl returns the latest armoctl version, using the
// on-disk cache if it is fresher than CacheTTL. On fetch failure with
// a stale cache present, the stale value is returned so the update
// banner keeps working offline — same fallback behaviour as
// GetLatestVersions for the multi-agent path.
func GetLatestArmoctl() (string, error) {
	cached := loadArmoctlCache()
	if cached != nil && time.Since(cached.FetchedAt) <= CacheTTL {
		return cached.Version, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), FetchTimeout)
	defer cancel()

	v, err := FetchLatestArmoctl(ctx)
	if err != nil {
		if cached != nil {
			return cached.Version, nil
		}
		return "", err
	}
	_ = saveArmoctlCache(v)
	return v, nil
}
