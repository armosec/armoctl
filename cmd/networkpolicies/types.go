package networkpolicies

var PolicySummary = []string{"name", "namespace", "cluster", "kind", "creationTimestamp"}

type Field struct {
	Name string
	Doc  string
}

func Cheatsheet() []Field {
	return []Field{
		{"name", "Policy name."},
		{"namespace", "Kubernetes namespace."},
		{"cluster", "Cluster name."},
		{"kind", "NetworkPolicy kind."},
		{"creationTimestamp", "RFC3339 creation."},
	}
}
