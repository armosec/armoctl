package runtimerules

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-runtime-rules",
		Cluster: "runtime-rules",
		Description: "Runtime detection rules — CRUD on the per-rule policy surface (the ARMO equivalent " +
			"of a Falco rule). Use to add, modify, or evaluate runtime rules.",
		Summary: "A rule is the smallest unit of runtime detection: 'fire when X happens.' Rules are " +
			"bundled into runtime policies (next cluster). Custom rules have ruleType 'Custom'; " +
			"ARMO-managed rules have ruleType 'Managed' and cannot be deleted.",
		FieldNotes: map[string]string{
			"ruleType": "Managed | Custom. Managed rules are maintained by ARMO and cannot be deleted. " +
				"Custom rules are user-created and fully mutable.",
			"policyTypes": "Policy type categories this rule belongs to (e.g. ADR, CDR). Used to group " +
				"rules into policy bundles.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "Create a rule from a JSON file",
				Body:  "```\narmoctl runtime-rules create --name 'my-rule' --rule-file rule.json --dry-run\n```",
			},
		},
	})
}

func convertCheatsheet(in []Field) map[string][]skillmeta.Field {
	fs := make([]skillmeta.Field, len(in))
	for i, f := range in {
		fs[i] = skillmeta.Field{Name: f.Name, Doc: f.Doc}
	}
	return map[string][]skillmeta.Field{"runtime-rules": fs}
}
