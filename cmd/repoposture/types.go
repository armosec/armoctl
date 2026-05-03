// Package repoposture implements the `armoctl repo-posture` cluster.
package repoposture

var RepositorySummary = []string{"name", "owner", "provider", "branch", "lastScanTime", "complianceScore", "failedControlsCount"}
var FileSummary = []string{"path", "name", "type", "lastScanTime", "complianceScore", "failedControlsCount"}
var ResourceSummary = []string{"name", "kind", "filePath", "complianceScore", "failedControlsCount"}
var FailedControlSummary = []string{"id", "name", "severity", "framework", "scoreFactor", "complianceScore"}

type Field struct {
	Name string
	Doc  string
}

func Cheatsheet() map[string][]Field {
	return map[string][]Field{
		"repositories": {
			{"name", "Repository name."},
			{"owner", "Owner / organization."},
			{"provider", "git provider (github, gitlab, ...)."},
			{"branch", "Default branch."},
			{"lastScanTime", "RFC3339 last scan."},
			{"complianceScore", "0..1 compliance score."},
			{"failedControlsCount", "Failed control count for the repo."},
		},
		"files": {
			{"path", "Path within the repo."},
			{"name", "File name."},
			{"type", "File type (yaml, terraform, ...)."},
			{"lastScanTime", "RFC3339 last scan."},
			{"complianceScore", "0..1 compliance score."},
			{"failedControlsCount", "Failed control count for the file."},
		},
		"resources": {
			{"name", "Resource name."},
			{"kind", "Kind (Deployment, Pod, ...)."},
			{"filePath", "Source file path."},
			{"complianceScore", "0..1 compliance score."},
			{"failedControlsCount", "Failed control count for the resource."},
		},
		"failed-controls": {
			{"id", "Control ID."},
			{"name", "Control name."},
			{"severity", "Severity bucket."},
			{"framework", "Owning framework."},
			{"scoreFactor", "Weighting factor."},
			{"complianceScore", "Per-control compliance 0..1."},
		},
	}
}
