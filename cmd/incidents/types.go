// Package incidents implements the `armoctl incidents` cluster.
package incidents

// SummaryFields is the default projection applied to `incidents list`.
var SummaryFields = []string{
	"guid", "name", "attributes.incidentStatus",
	"updatedTime", "clusterName",
	"designators.wlid", "cloudMetadata.region", "kind",
}

// Field is one entry in the per-resource cheatsheet.
type Field struct {
	Name string
	Doc  string
}

// Cheatsheet returns the curated field list used both for `armoctl incidents fields`
// and for the auto-generated section in SKILL.md.
func Cheatsheet() []Field {
	return []Field{
		{"guid", "Stable incident ID; primary key for resolve/explain/alerts."},
		{"name", "Short rule/incident name (e.g. \"Suspicious binary execution\")."},
		{"kind", "Incident kind/category (e.g. \"ThreatDetection\")."},
		{"attributes.incidentStatus", "Current status: Open | Investigating | Dismissed | Resolved. Access with path syntax."},
		{"updatedTime", "RFC3339 timestamp of the last status change."},
		{"timestamp", "RFC3339 time the incident was first raised."},
		{"clusterName", "Kubernetes cluster that reported the incident."},
		{"designators.wlid", "Workload ID (ARMO wlid format). Access with path syntax."},
		{"cloudMetadata.region", "Cloud region where the workload runs. Access with path syntax."},
		{"signature", "Unique fingerprint identifying the rule that fired."},
	}
}
