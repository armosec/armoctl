// Package risks implements the `armoctl risks` cluster.
package risks

var RiskSummary = []string{"name", "id", "severity", "category", "controlID", "smartRemediation", "creationTime"}
var ResourceSummary = []string{"name", "namespace", "kind", "cluster", "severity", "riskCount"}

type Field struct {
	Name string
	Doc  string
}

func Cheatsheet() map[string][]Field {
	return map[string][]Field{
		"risks": {
			{"name", "Risk name."},
			{"id", "Risk ID."},
			{"severity", "Severity bucket."},
			{"category", "Risk category."},
			{"controlID", "Owning control ID."},
			{"smartRemediation", "Whether smart remediation is available."},
			{"creationTime", "RFC3339 first-seen time."},
		},
		"resources": {
			{"name", "Resource name."},
			{"namespace", "Kubernetes namespace."},
			{"kind", "Resource kind."},
			{"cluster", "Cluster name."},
			{"severity", "Highest severity for the resource."},
			{"riskCount", "Number of risks affecting the resource."},
		},
	}
}
