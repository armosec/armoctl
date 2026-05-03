package vulns

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-vulns",
		Cluster: "vulns",
		Description: "ARMO vulnerability triage — list CVEs, find affected images/hosts/workloads, " +
			"check runtime relevance, manage exception policies. Use when the user is investigating " +
			"package vulnerabilities, container CVEs, or remediation prioritization.",
		Summary: "The `vulns` cluster covers the runtime + scan vulnerability surface. The most " +
			"important triage axis is `isRelevant`: ARMO observes which packages are actually loaded " +
			"in running workloads, so a Critical CVE in dormant code is a much lower priority " +
			"than the same CVE in an in-use library. Always filter by isRelevant when scoping urgent work.",
		FieldNotes: map[string]string{
			"isRelevant":  "Runtime-loaded vs. dormant on disk. Critical for triage: a Critical CVE in dormant code is much lower priority than the same CVE in an in-use library. Filter with `--query '.items[] | select(.attributes.isRelevant == true)'`.",
			"fixVersions": "Versions that fix known CVEs. Empty means no fix available upstream — don't suggest 'upgrade' as a remediation in that case.",
			"severity":    "ARMO severity (critical | high | medium | low | unknown), not raw CVSS — already adjusted for runtime context and exception policies.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "Critical CVEs that are actually in use",
				Body:  "```\narmoctl vulns cves --severity Critical --query '.items[] | select(.attributes.isRelevant == true)'\n```",
			},
			{
				Title: "List exceptions for a CVE",
				Body:  "```\narmoctl vulns exceptions list --cve CVE-2024-12345\n```",
			},
		},
	})
}

func convertCheatsheet(in map[string][]Field) map[string][]skillmeta.Field {
	out := make(map[string][]skillmeta.Field, len(in))
	for k, v := range in {
		fs := make([]skillmeta.Field, len(v))
		for i, f := range v {
			fs[i] = skillmeta.Field{Name: f.Name, Doc: f.Doc}
		}
		out[k] = fs
	}
	return out
}
