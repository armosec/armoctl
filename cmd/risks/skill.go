package risks

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-risks",
		Cluster: "risks",
		Description: "Security risks (cross-cutting risk view) — list/resources/severities and exception " +
			"policies. Use when working with the unified ARMO risk score, not per-domain CVE/posture findings.",
		Summary: "Risks are the unified prioritisation surface that combines vulnerability + posture + " +
			"runtime signal into a single severity per (resource × risk-class). Exception policies live " +
			"here too.",
		FieldNotes: map[string]string{
			"severity": "Composite severity — already accounts for runtime context, exceptions, and " +
				"exposure. Use to filter with --severity Critical|high|medium|low.",
			"policyIDs": "On exceptions: the risk IDs the exception applies to. Single-element in " +
				"current API even though it's an array.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "List Critical risks",
				Body:  "```\narmoctl risks list --severity Critical\n```",
			},
			{
				Title: "Create an exception for a risk with an expiry date",
				Body: "```\narmoctl risks exceptions create --risk-id <id> " +
					"--reason 'planned remediation' --expires 2026-06-01T00:00:00Z --dry-run\n```",
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
