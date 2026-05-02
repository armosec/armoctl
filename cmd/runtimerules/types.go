package runtimerules

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

// RuleSummary is the default projection for `runtime-rules list`.
var RuleSummary = []string{"guid", "name", "description", "ruleType", "createdBy", "creationTime", "policyTypes"}

// Field is one entry in the per-resource cheatsheet.
type Field struct {
	Name string
	Doc  string
}

// Cheatsheet returns the curated field list used both for `armoctl runtime-rules fields`
// and for documentation.
func Cheatsheet() []Field {
	return []Field{
		{"guid", "Stable rule ID."},
		{"name", "Rule name."},
		{"description", "Human-readable description."},
		{"ruleType", "Managed | Custom."},
		{"createdBy", "Author."},
		{"creationTime", "RFC3339 creation."},
		{"policyTypes", "ADR | CDR etc."},
		{"rule", "Full rule expression (in --full)."},
	}
}
