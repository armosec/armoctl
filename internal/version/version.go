package version

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/mod/semver"
)

const (
	// DistributionURL is the CloudFront URL for armoctl distribution.
	DistributionURL = "https://package-distribution.armosec.io/armoctl"

	// VersionEndpoint is the path to the version.json file.
	VersionEndpoint = "/version.json"

	// FetchTimeout is the timeout for fetching version info.
	FetchTimeout = 5 * time.Second

	// MaxResponseSize is the maximum size of the version.json response (1MB).
	MaxResponseSize = 1024 * 1024

	// DefaultAgentImageRepo is the ECR repository for the ptrace agent image.
	// The %s placeholder is for the tag/version.
	DefaultAgentImageRepo = "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:%s"

	// DefaultOperatorImageRepo is the ECR repository for the operator image.
	// The first %s is for region, second %s is for tag/version.
	DefaultOperatorImageRepo = "015253967648.dkr.ecr.%s.amazonaws.com/ecs-operator:%s"

	// FallbackTag is used when version info is not available.
	FallbackTag = "latest"
)

// Versions holds the latest versions of all components.
type Versions struct {
	Armoctl     string `json:"armoctl"`
	Operator    string `json:"operator"`
	PtraceAgent string `json:"ptrace-agent"`
}

// FetchLatest fetches the latest version information from CloudFront.
func FetchLatest() (*Versions, error) {
	return FetchLatestWithContext(context.Background())
}

// FetchLatestWithContext fetches the latest version information with context support.
func FetchLatestWithContext(ctx context.Context) (*Versions, error) {
	return FetchLatestFromURL(ctx, DistributionURL+VersionEndpoint)
}

// FetchLatestFromURL fetches version information from a specific URL.
// This is useful for testing with mock servers.
func FetchLatestFromURL(ctx context.Context, url string) (*Versions, error) {
	client := &http.Client{Timeout: FetchTimeout}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching version info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Limit response size to prevent OOM from malicious servers
	limitedReader := io.LimitReader(resp.Body, MaxResponseSize)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var versions Versions
	if err := json.Unmarshal(body, &versions); err != nil {
		return nil, fmt.Errorf("parsing version info: %w", err)
	}

	return &versions, nil
}

// UpdateInfo contains information about available updates.
type UpdateInfo struct {
	ArmoCtlCurrent string
	ArmoCtlLatest  string
	HasUpdate      bool
}

// CheckForUpdates compares the current version against the latest.
func CheckForUpdates(currentVersion string, latest *Versions) *UpdateInfo {
	// Handle nil latest gracefully
	if latest == nil {
		return &UpdateInfo{
			ArmoCtlCurrent: currentVersion,
			HasUpdate:      false,
		}
	}

	info := &UpdateInfo{
		ArmoCtlCurrent: currentVersion,
		ArmoCtlLatest:  latest.Armoctl,
	}

	// Skip update check for dev builds
	if currentVersion == "dev" || currentVersion == "" {
		return info
	}

	// Use proper semver comparison
	// semver.Compare returns -1 if v < w, 0 if v == w, +1 if v > w
	// We want to show update if current < latest
	if semver.IsValid(currentVersion) && semver.IsValid(latest.Armoctl) {
		if semver.Compare(currentVersion, latest.Armoctl) < 0 {
			info.HasUpdate = true
		}
	} else {
		// Fallback to string comparison if versions are not valid semver
		// This handles cases like pre-release versions
		if currentVersion != latest.Armoctl {
			info.HasUpdate = true
		}
	}

	return info
}

// GetAgentImage returns the ptrace agent image with the latest tag from cache.
// Falls back to "latest" tag if version info is not available.
func GetAgentImage() string {
	tag := FallbackTag

	cached := LoadCache()
	if cached != nil && cached.Versions.PtraceAgent != "" {
		tag = cached.Versions.PtraceAgent
	}

	return fmt.Sprintf(DefaultAgentImageRepo, tag)
}

// GetOperatorImage returns the operator image for a region with the latest tag from cache.
// Falls back to "latest" tag if version info is not available.
func GetOperatorImage(region string) string {
	tag := FallbackTag

	cached := LoadCache()
	if cached != nil && cached.Versions.Operator != "" {
		tag = cached.Versions.Operator
	}

	return fmt.Sprintf(DefaultOperatorImageRepo, region, tag)
}
