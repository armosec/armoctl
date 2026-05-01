// Package attackchains implements the `armoctl attack-chains` cluster.
package attackchains

var ChainSummary = []string{"name", "guid", "creationTime", "severity", "clusterName", "namespace"}

type Field struct {
	Name string
	Doc  string
}

func Cheatsheet() []Field {
	return []Field{
		{"name", "Attack chain name."},
		{"guid", "Stable identifier."},
		{"creationTime", "RFC3339 first-seen time."},
		{"severity", "Severity bucket."},
		{"clusterName", "Cluster name."},
		{"namespace", "Kubernetes namespace."},
	}
}
