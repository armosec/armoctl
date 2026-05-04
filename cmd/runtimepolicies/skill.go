package runtimepolicies

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-runtime-policies",
		Cluster: "runtime-policies",
		Description: "Runtime policies — bundles of rules attached to clusters/namespaces/workloads. " +
			"Use to manage which detection rules apply where.",
		Summary: "A policy is a bag of runtime-rules with a binding scope (cluster, namespace, workload). " +
			"When a workload runs, the union of policies that bind to it determines which rules evaluate.",
		FieldNotes: map[string]string{
			"scope": "Scope object describing which clusters/namespaces/workloads this policy binds to. " +
				"Most-specific binding wins on conflict.",
			"enabled": "Whether the policy is currently active. Disabled policies are stored but do " +
				"not generate incidents.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "List managed runtime policies",
				Body:  "```\narmoctl runtime-policies list --rulesettype Managed\n```",
			},
			{
				Title: "List custom runtime policies",
				Body:  "```\narmoctl runtime-policies list --rulesettype Custom\n```",
			},
		},
	})
}

func convertCheatsheet(in []Field) map[string][]skillmeta.Field {
	fs := make([]skillmeta.Field, len(in))
	for i, f := range in {
		fs[i] = skillmeta.Field{Name: f.Name, Doc: f.Doc}
	}
	return map[string][]skillmeta.Field{"runtime-policies": fs}
}
