package version

import (
	"time"

	"golang.org/x/mod/semver"
)

const (
	// DistributionURL is the CloudFront URL for armoctl binary distribution.
	DistributionURL = "https://package-distribution.armosec.io/armoctl"

	// FetchTimeout is the timeout for fetching version info.
	FetchTimeout = 5 * time.Second
)

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

	cur := ensureVPrefix(currentVersion)
	lat := ensureVPrefix(latestVersion)

	if semver.IsValid(cur) && semver.IsValid(lat) {
		if semver.Compare(cur, lat) < 0 {
			info.HasUpdate = true
		}
	} else if cur != lat {
		info.HasUpdate = true
	}

	return info
}

func ensureVPrefix(v string) string {
	if v != "" && v[0] != 'v' {
		return "v" + v
	}
	return v
}
