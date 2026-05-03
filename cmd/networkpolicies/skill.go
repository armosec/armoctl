package networkpolicies

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-network-policies",
		Cluster: "network-policies",
		Description: "Generated NetworkPolicies — list discovered policies and generate one for a workload " +
			"from observed traffic. Use to harden cluster network egress/ingress.",
		Summary: "ARMO observes runtime traffic and emits a least-privilege NetworkPolicy YAML for any " +
			"selected workload. List shows historical policies; generate produces one on-demand.",
		FieldNotes: map[string]string{
			"kind": "NetworkPolicy kind. Use 'inventory unique-values kind' to verify the workload " +
				"kind name before generating a policy for it.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "Generate a network policy for a workload",
				Body:  "```\narmoctl network-policies generate --wlid <workload-id>\n```",
			},
		},
	})
}

func convertCheatsheet(in []Field) map[string][]skillmeta.Field {
	fs := make([]skillmeta.Field, len(in))
	for i, f := range in {
		fs[i] = skillmeta.Field{Name: f.Name, Doc: f.Doc}
	}
	return map[string][]skillmeta.Field{"policies": fs}
}
