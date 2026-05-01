// Package inventory implements the `armoctl inventory` cluster.
package inventory

var WorkloadSummary = []string{"wlid", "name", "namespace", "kind", "cluster", "lastInventoryScanTime"}

type Field struct {
	Name string
	Doc  string
}

func Cheatsheet() []Field {
	return []Field{
		{"wlid", "Workload ID."},
		{"name", "Workload name."},
		{"namespace", "Kubernetes namespace."},
		{"kind", "Resource kind."},
		{"cluster", "Cluster name."},
		{"lastInventoryScanTime", "RFC3339 last scan time."},
	}
}
