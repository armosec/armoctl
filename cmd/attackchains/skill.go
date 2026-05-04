package attackchains

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-attack-chains",
		Cluster: "attack-chains",
		Description: "Attack chains — multi-step kill-chain views built by ARMO from runtime + posture " +
			"signal. Use when the user wants to understand how vulnerabilities chain into reachable exploit paths.",
		Summary: "An attack chain links a posture weakness, a vulnerable component, and runtime context " +
			"into a sequence an attacker could traverse. List view shows the highest-severity chains; " +
			"details show the per-step evidence.",
		FieldNotes: map[string]string{
			"severity": "Chain severity — reflects the worst-case step in the chain. Use to prioritise " +
				"which chains to investigate first.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "List active attack chains",
				Body:  "```\narmoctl attack-chains list\n```",
			},
		},
	})
}

func convertCheatsheet(in []Field) map[string][]skillmeta.Field {
	fs := make([]skillmeta.Field, len(in))
	for i, f := range in {
		fs[i] = skillmeta.Field{Name: f.Name, Doc: f.Doc}
	}
	return map[string][]skillmeta.Field{"attack-chains": fs}
}
