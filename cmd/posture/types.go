// Package posture implements the `armoctl posture` cluster.
package posture

var FrameworkSummary = []string{"name", "complianceScore", "totalControls", "failedControls", "skippedControls", "passedControls", "lastRun"}
var ControlSummary = []string{"name", "id", "severity", "complianceScore", "framework", "status", "scoreFactor"}
var ResourceSummary = []string{"name", "namespace", "kind", "cluster", "complianceScore", "failedControlsCount", "warningControlsCount", "totalControlsCount"}
var ExceptionSummary = []string{"guid", "name", "policyType", "creationTime", "actions"}

type Field struct {
	Name string
	Doc  string
}

func Cheatsheet() map[string][]Field {
	return map[string][]Field{
		"frameworks": {
			{"name", "Framework name (e.g. NSA, MITRE)."},
			{"complianceScore", "Aggregate compliance score 0..1."},
			{"totalControls", "Total controls in the framework."},
			{"failedControls", "Failed control count."},
			{"passedControls", "Passed control count."},
			{"skippedControls", "Skipped control count."},
			{"lastRun", "RFC3339 last scan time."},
		},
		"controls": {
			{"name", "Control name."},
			{"id", "Control ID (e.g. C-0001)."},
			{"severity", "Severity bucket."},
			{"complianceScore", "Per-control compliance 0..1."},
			{"framework", "Owning framework."},
			{"status", "passed | failed | skipped."},
			{"scoreFactor", "Weighting factor for scoring."},
		},
		"resources": {
			{"name", "Resource name."},
			{"namespace", "Kubernetes namespace."},
			{"kind", "Resource kind."},
			{"cluster", "Cluster name."},
			{"complianceScore", "0..1 compliance score."},
			{"failedControlsCount", "Failing control count for this resource."},
			{"warningControlsCount", "Warning control count."},
			{"totalControlsCount", "Total controls evaluated."},
		},
		"exceptions": {
			{"guid", "Policy GUID."},
			{"name", "Policy name."},
			{"policyType", "postureExceptionPolicy."},
			{"creationTime", "RFC3339 creation."},
			{"actions", "[\"alertOnly\"]."},
		},
	}
}
