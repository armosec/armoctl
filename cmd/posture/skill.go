package posture

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-posture",
		Cluster: "posture",
		Description: "Kubernetes posture scanning — controls, frameworks, exceptions. Use when assessing " +
			"compliance posture (NSA, MITRE, etc.) or managing posture exception policies.",
		Summary: "Posture is config-time scanning of K8s resources against control frameworks. A 'failed " +
			"control' means a resource violates a rule from a framework like NSA-CISA. Exception policies " +
			"suppress specific (control × resource) pairs.",
		FieldNotes: map[string]string{
			"id": "Stable control identifier (e.g. C-0001). Prefer this over name when scripting — " +
				"names can change between framework versions.",
			"framework": "Owning framework name (NSA, MITRE, ArmoBest, etc.). A single control can " +
				"belong to several frameworks.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "List controls for a specific framework",
				Body:  "```\narmoctl posture controls --framework NSA\n```",
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
