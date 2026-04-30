// Package incidents implements the `armoctl incidents` cluster.
package incidents

// SummaryFields is the default projection applied to `incidents list`.
var SummaryFields = []string{
	"guid", "name", "severity", "status", "creationTimestamp",
	"resource.cluster", "resource.namespace", "resource.workload",
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
		{"guid", "Stable incident ID; primary key for get/resolve/explain."},
		{"name", "Short rule/incident name (e.g. \"Suspicious binary\")."},
		{"severity", "critical | high | medium | low."},
		{"status", "open | resolved | investigating."},
		{"creationTimestamp", "RFC3339 time the incident was raised."},
		{"resource.cluster", "Cluster the workload belongs to."},
		{"resource.namespace", "Kubernetes namespace (or N/A for ECS)."},
		{"resource.workload", "Workload name (deployment/service/task)."},
		{"alertCount", "Number of alerts grouped under this incident."},
		{"resolvedBy", "User/service that resolved the incident, if any."},
		{"resolutionReason", "Free-text reason recorded at resolve time."},
	}
}
