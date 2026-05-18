package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCheckForUpdates(t *testing.T) {
	tests := []struct {
		name       string
		current    string
		latest     string
		wantUpdate bool
	}{
		{"same version", "v0.0.42", "v0.0.42", false},
		{"new patch version available", "v0.0.40", "v0.0.42", true},
		{"new minor version available", "v0.1.0", "v0.2.0", true},
		{"new major version available", "v1.0.0", "v2.0.0", true},
		{"dev version - no update", "dev", "v0.0.42", false},
		{"empty current - no update", "", "v0.0.42", false},
		{"empty latest - no update (fetch failed)", "v0.0.40", "", false},
		{"current newer than latest (beta user)", "v0.0.50", "v0.0.42", false},
		{"semver comparison v0.0.9 vs v0.0.10", "v0.0.9", "v0.0.10", true},
		{"semver comparison v0.9.0 vs v0.10.0", "v0.9.0", "v0.10.0", true},
		{"pre-release version", "v0.0.42-rc1", "v0.0.42", true},
		{"goreleaser strips v prefix - same version", "0.0.11", "v0.0.11", false},
		{"goreleaser strips v prefix - update available", "0.0.10", "v0.0.11", true},
		{"both without v prefix - same version", "0.0.11", "0.0.11", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := CheckForUpdates(tt.current, tt.latest)
			if info.HasUpdate != tt.wantUpdate {
				t.Errorf("CheckForUpdates() HasUpdate = %v, want %v", info.HasUpdate, tt.wantUpdate)
			}
			if info.ArmoCtlCurrent != tt.current {
				t.Errorf("ArmoCtlCurrent = %v, want %v", info.ArmoCtlCurrent, tt.current)
			}
			if info.ArmoCtlLatest != tt.latest {
				t.Errorf("ArmoCtlLatest = %v, want %v", info.ArmoCtlLatest, tt.latest)
			}
		})
	}
}

func TestFetchLatestArmoctlFrom(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		status    int
		want      string
		wantError bool
	}{
		{name: "happy path", body: "v0.0.10\n", status: 200, want: "v0.0.10"},
		{name: "no trailing newline", body: "v1.2.3", status: 200, want: "v1.2.3"},
		{name: "pre-release tag", body: "v0.0.10-rc1\n", status: 200, want: "v0.0.10-rc1"},
		{name: "rejects HTML SPA fallthrough", body: "<!DOCTYPE html><html>", status: 200, wantError: true},
		{name: "rejects empty body", body: "", status: 200, wantError: true},
		{name: "rejects 404", body: "", status: 404, wantError: true},
		{name: "rejects 500", body: "internal error", status: 500, wantError: true},
		{name: "rejects JSON-shaped body without angle brackets", body: `{"version":"v0.0.10"}`, status: 200, wantError: true},
		{name: "rejects bare integer", body: "1\n", status: 200, wantError: true},
		{name: "rejects unprefixed version", body: "0.0.10\n", status: 200, wantError: true},
		{name: "rejects BOM-prefixed valid version", body: "\xef\xbb\xbfv0.0.10\n", status: 200, wantError: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()

			got, err := FetchLatestArmoctlFrom(context.Background(), srv.URL)
			if tc.wantError {
				if err == nil {
					t.Errorf("FetchLatestArmoctlFrom() = %q, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("FetchLatestArmoctlFrom() unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("FetchLatestArmoctlFrom() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFetchLatestArmoctlFrom_SizeCap(t *testing.T) {
	// Server sends a 1 MiB body. The reader caps at maxLatestTxtBytes
	// (64), and the truncated bytes ("aaaa…") fail semver validation
	// — we want to confirm the cap fires (no OOM) and the validator
	// rejects the garbage rather than us returning a bogus version.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("a", 1024*1024)))
	}))
	defer srv.Close()

	_, err := FetchLatestArmoctlFrom(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error: 1MiB of 'a' should fail semver validation after truncation")
	}
}

// writeArmoctlCache writes a cache file with the supplied age.
// Negative age = future-dated (treated as fresh); positive = how far
// in the past, e.g. (2 * CacheTTL) to make the entry stale.
func writeArmoctlCache(t *testing.T, version string, age time.Duration) {
	t.Helper()
	if err := EnsureCacheDir(); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(ArmoctlVersionCache{
		FetchedAt: time.Now().Add(-age),
		Version:   version,
	})
	if err != nil {
		t.Fatal(err)
	}
	path, err := armoctlCachePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestGetLatestArmoctlWith_FreshCacheServed(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeArmoctlCache(t, "v9.9.9", 0)

	called := false
	fetcher := func(_ context.Context) (string, error) {
		called = true
		return "v0.0.1", nil
	}
	got, err := getLatestArmoctlWith(fetcher)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v9.9.9" {
		t.Errorf("got %q, want v9.9.9 (from cache)", got)
	}
	if called {
		t.Error("fetcher should not have been called when cache was fresh")
	}
}

func TestGetLatestArmoctlWith_StaleCacheRefreshes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeArmoctlCache(t, "v0.0.1", 2*CacheTTL)

	fetcher := func(_ context.Context) (string, error) {
		return "v0.0.10", nil
	}
	got, err := getLatestArmoctlWith(fetcher)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v0.0.10" {
		t.Errorf("got %q, want v0.0.10 (from fetch)", got)
	}

	// Cache should now hold the new value with a fresh timestamp.
	cached := loadArmoctlCache()
	if cached == nil || cached.Version != "v0.0.10" {
		t.Errorf("cache after refresh = %+v, want Version=v0.0.10", cached)
	}
}

func TestGetLatestArmoctlWith_FallsBackToStaleOnFetchError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeArmoctlCache(t, "v0.0.1", 2*CacheTTL)

	fetcher := func(_ context.Context) (string, error) {
		return "", fmt.Errorf("network unreachable")
	}
	got, err := getLatestArmoctlWith(fetcher)
	if err != nil {
		t.Fatalf("expected stale fallback, got error: %v", err)
	}
	if got != "v0.0.1" {
		t.Errorf("got %q, want v0.0.1 (stale fallback)", got)
	}
}

func TestGetLatestArmoctlWith_NoCacheNoFetchPropagatesError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	fetcher := func(_ context.Context) (string, error) {
		return "", fmt.Errorf("network unreachable")
	}
	_, err := getLatestArmoctlWith(fetcher)
	if err == nil {
		t.Fatal("expected error when both cache and fetch are unavailable")
	}
}

func TestPadding(t *testing.T) {
	tests := []struct {
		current string
		latest  string
	}{
		{"v0.0.1", "v0.0.42"},
		{"v1.0.0", "v2.0.0"},
		{"dev", "v0.0.1"},
		{"v0.0.0", "v999.999.999"},
	}

	for _, tt := range tests {
		t.Run(tt.current+"->"+tt.latest, func(t *testing.T) {
			result := padding(tt.current, tt.latest)
			if len(result) == 0 {
				t.Error("padding() returned empty string")
			}
		})
	}
}


// --- ECS image lookup: read-only consumers of the legacy multi-agent cache ---

func writeMultiAgentCache(t *testing.T, v Versions) {
	t.Helper()
	if err := EnsureCacheDir(); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(CachedVersions{FetchedAt: time.Now(), Versions: v})
	if err != nil {
		t.Fatal(err)
	}
	path, err := CachePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, body, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestGetAgentImage_NoCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	image := GetAgentImage()
	expected := "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:latest"
	if image != expected {
		t.Errorf("GetAgentImage() = %v, want %v", image, expected)
	}
}

func TestGetAgentImage_WithCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeMultiAgentCache(t, Versions{PtraceAgent: "v3.0.0"})

	expected := "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:v3.0.0"
	if got := GetAgentImage(); got != expected {
		t.Errorf("GetAgentImage() = %v, want %v", got, expected)
	}
}

func TestGetOperatorImage_NoCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	expected := "015253967648.dkr.ecr.us-east-1.amazonaws.com/ecs-operator:latest"
	if got := GetOperatorImage("us-east-1"); got != expected {
		t.Errorf("GetOperatorImage() = %v, want %v", got, expected)
	}
}

func TestGetOperatorImage_WithCache(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeMultiAgentCache(t, Versions{ECSOperator: "v2.0.0"})

	expected := "015253967648.dkr.ecr.eu-west-1.amazonaws.com/ecs-operator:v2.0.0"
	if got := GetOperatorImage("eu-west-1"); got != expected {
		t.Errorf("GetOperatorImage() = %v, want %v", got, expected)
	}
}

func TestGetOperatorImage_DifferentRegions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	writeMultiAgentCache(t, Versions{ECSOperator: "v1.0.0"})

	regions := []string{"us-east-1", "us-west-2", "eu-north-1", "ap-southeast-1"}
	for _, region := range regions {
		t.Run(region, func(t *testing.T) {
			image := GetOperatorImage(region)
			expected := "015253967648.dkr.ecr." + region + ".amazonaws.com/ecs-operator:v1.0.0"
			if image != expected {
				t.Errorf("GetOperatorImage(%s) = %v, want %v", region, image, expected)
			}
		})
	}
}

