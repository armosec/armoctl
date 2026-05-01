// Package vulns implements the `armoctl vulns` cluster.
package vulns

// Per-scope summary projections. The vuln endpoints return very different shapes
// per scope, so each list command picks the right SummaryFields.

var WorkloadSummary = []string{
	"wlid", "name", "namespace", "kind", "cluster",
	"lastScanTime", "imagesCount",
	"criticalCount", "highCount", "mediumCount", "lowCount",
}

var ImageSummary = []string{
	"repository", "tag", "registry", "digest",
	"lastScanTime", "criticalCount", "highCount", "mediumCount", "lowCount",
}

var ComponentSummary = []string{
	"name", "version", "packageType",
	"criticalCount", "highCount", "mediumCount", "lowCount",
	"workloadsCount", "imagesCount", "hasHotCVE",
}

var CVESummary = []string{
	"name", "id", "severity", "severityScore",
	"exploitable", "isRelevant", "discoveredDate",
}

var HostSummary = []string{
	"hostName", "hostType", "accountName", "region", "kernelVersion",
}

// Field is one entry in a per-resource cheatsheet.
type Field struct {
	Name string
	Doc  string
}

// Cheatsheet returns curated cheatsheets per scope.
func Cheatsheet() map[string][]Field {
	return map[string][]Field{
		"workloads": {
			{"wlid", "Workload ID; primary identifier."},
			{"name", "Workload name (deployment / statefulset / etc.)."},
			{"namespace", "Kubernetes namespace."},
			{"kind", "Resource kind."},
			{"cluster", "Cluster name."},
			{"lastScanTime", "RFC3339 time of the most recent scan."},
			{"imagesCount", "Number of images in the workload."},
			{"criticalCount", "Count of critical CVEs across the workload."},
			{"highCount", "Count of high CVEs."},
			{"mediumCount", "Count of medium CVEs."},
			{"lowCount", "Count of low CVEs."},
		},
		"images": {
			{"repository", "Image repository."},
			{"tag", "Image tag."},
			{"registry", "Image registry."},
			{"digest", "Image digest."},
			{"lastScanTime", "RFC3339 time of the most recent scan."},
			{"criticalCount", "Count of critical CVEs in the image."},
			{"highCount", "Count of high CVEs."},
			{"mediumCount", "Count of medium CVEs."},
			{"lowCount", "Count of low CVEs."},
			{"clusters", "Clusters where this image runs."},
			{"namespaces", "Namespaces where this image runs."},
		},
		"components": {
			{"name", "Component (package) name."},
			{"version", "Component version."},
			{"packageType", "Package type (go-module, npm, ...)."},
			{"fixVersions", "Versions that fix known CVEs."},
			{"criticalCount", "Count of critical CVEs."},
			{"highCount", "Count of high CVEs."},
			{"workloadsCount", "Workloads using this component."},
			{"imagesCount", "Images containing this component."},
			{"hasHotCVE", "Whether the component has a 'hot' CVE."},
		},
		"cves": {
			{"name", "CVE name (e.g. GHSA-... or CVE-...)."},
			{"id", "Stable identifier."},
			{"severity", "critical | high | medium | low | unknown."},
			{"severityScore", "Numeric severity score."},
			{"exploitable", "Known to be exploitable in the wild."},
			{"isRelevant", "ARMO relevance signal: actually loaded at runtime."},
			{"discoveredDate", "RFC3339 first-seen time."},
			{"componentInfo", "Component context (name+version)."},
			{"cvssInfo", "Full CVSS info."},
		},
		"hosts": {
			{"hostName", "Host name."},
			{"hostType", "Host type (kubernetes/ec2/...)."},
			{"accountName", "Cloud account name."},
			{"region", "Cloud region."},
			{"kernelVersion", "Kernel version of the host."},
		},
	}
}
