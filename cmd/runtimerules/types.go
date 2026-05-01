package runtimerules

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
