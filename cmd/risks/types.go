// Package risks implements the `armoctl risks` cluster.
package risks

var RiskSummary = []string{"name", "id", "severity", "category", "controlID", "smartRemediation", "creationTime"}
var ResourceSummary = []string{"name", "namespace", "kind", "cluster", "severity", "riskCount"}
var ExceptionSummary = []string{"guid", "name", "policyIDs", "reason", "expirationDate", "creationTime", "createdBy"}

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
		"exceptions": {
			{"guid", "Exception policy GUID; required for get/update/delete."},
			{"name", "Human-readable policy name."},
			{"policyIDs", "Security risk IDs covered by the exception (exactly one supported)."},
			{"reason", "Reason recorded when the risk was accepted."},
			{"expirationDate", "RFC3339 expiration; null = no expiry."},
			{"creationTime", "RFC3339 first-created time."},
			{"createdBy", "User that created the policy."},
			{"resources", "Optional resource scope (PortalDesignators)."},
		},
	}
}
