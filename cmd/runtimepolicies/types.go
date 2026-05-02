package runtimepolicies

import "github.com/armosec/armoctl/internal/clierr"

// codeForStatus returns the appropriate clierr.Code for an HTTP status code.
func codeForStatus(s int) clierr.Code {
	switch {
	case s == 401, s == 403:
		return clierr.CodeAuth
	case s == 404:
		return clierr.CodeNotFound
	case s == 409:
		return clierr.CodeConflict
	case s >= 400 && s < 500:
		return clierr.CodeBadInput
	default:
		return clierr.CodeServer
	}
}

// PolicySummary is the default projection for `runtime-policies list`.
var PolicySummary = []string{"guid", "name", "description", "enabled", "scope", "creationTime"}

// Field is one entry in the per-resource cheatsheet.
type Field struct {
	Name string
	Doc  string
}

// Cheatsheet returns the curated field list used both for `armoctl runtime-policies fields`
// and for documentation.
func Cheatsheet() []Field {
	return []Field{
		{"guid", "Stable policy ID."},
		{"name", "Policy name."},
		{"description", "Human-readable description."},
		{"enabled", "Whether the policy is active."},
		{"scope", "Scope object (clusters/namespaces/workloads)."},
		{"creationTime", "RFC3339 creation."},
	}
}
