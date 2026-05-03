package seccomp

var ProfileSummary = []string{"name", "namespace", "cluster", "kind", "containerName"}

type Field struct {
	Name string
	Doc  string
}

func Cheatsheet() []Field {
	return []Field{
		{"name", "Profile name."},
		{"namespace", "Kubernetes namespace."},
		{"cluster", "Cluster name."},
		{"kind", "Resource kind."},
		{"containerName", "Container the profile applies to."},
	}
}
