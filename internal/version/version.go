package version

import (
	"fmt"
	"time"

	"golang.org/x/mod/semver"
)

const (
	// DistributionURL is the CloudFront URL for armoctl binary distribution.
	DistributionURL = "https://package-distribution.armosec.io/armoctl"

	// FetchTimeout is the timeout for fetching version info.
	FetchTimeout = 5 * time.Second

	// DefaultAgentImageRepo is the ECR repository for the ptrace agent image.
	// The %s placeholder is for the tag/version.
	DefaultAgentImageRepo = "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:%s"

	// DefaultOperatorImageRepo is the ECR repository for the operator image.
	// The first %s is for region, second %s is for tag/version.
	DefaultOperatorImageRepo = "015253967648.dkr.ecr.%s.amazonaws.com/ecs-operator:%s"

	// FallbackTag is used when version info is not available.
	FallbackTag = "latest"
)

// Versions holds component version strings cached on disk for the ECS
// image helpers (GetAgentImage / GetOperatorImage). The fetch flow that
// used to populate this struct from the cadashboardbe API has been
// removed — armoctl's own update check now reads from the binary
// distribution CDN directly. The struct itself is preserved so the
// existing on-disk cache file can still be read by ECS callers, which
// fall back to FallbackTag when no cached value is available.
type Versions struct {
	HostAgent   string `json:"hostAgent"`
	NodeAgent   string `json:"nodeAgent"`
	ECSAgent    string `json:"ecsAgent"`
	ECSOperator string `json:"ecsOperator"`
	Armoctl     string `json:"armoctl"`
	PtraceAgent string `json:"ptraceAgent"`
}

// UpdateInfo contains information about available updates.
type UpdateInfo struct {
	ArmoCtlCurrent string
	ArmoCtlLatest  string
	HasUpdate      bool
}

// CheckForUpdates compares the current armoctl version against the
// latest version string fetched from the CDN. Empty latest means the
// fetch failed and we have nothing to compare against — callers see
// HasUpdate=false and decide whether to surface a banner.
func CheckForUpdates(currentVersion, latestVersion string) *UpdateInfo {
	info := &UpdateInfo{
		ArmoCtlCurrent: currentVersion,
		ArmoCtlLatest:  latestVersion,
	}

	if latestVersion == "" {
		return info
	}

	// Skip update check for dev builds
	if currentVersion == "dev" || currentVersion == "" {
		return info
	}

	// Use proper semver comparison
	if semver.IsValid(currentVersion) && semver.IsValid(latestVersion) {
		if semver.Compare(currentVersion, latestVersion) < 0 {
			info.HasUpdate = true
		}
	} else if currentVersion != latestVersion {
		// Fallback to string comparison if versions are not valid semver
		info.HasUpdate = true
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
	if cached != nil && cached.Versions.ECSOperator != "" {
		tag = cached.Versions.ECSOperator
	}

	return fmt.Sprintf(DefaultOperatorImageRepo, region, tag)
}
