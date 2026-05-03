package repoposture

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-repo-posture",
		Cluster: "repo-posture",
		Description: "Repository posture — IaC scanning of a connected git repo for config issues, with " +
			"per-file and per-control views. Use when reviewing posture findings tied to a repo, not a live cluster.",
		Summary: "Same control surface as cluster posture, but the resources are files in a connected git " +
			"repo. Findings carry both file path and control ID, so they can be deep-linked back to the IaC source.",
		FieldNotes: map[string]string{
			"filePath": "Repo-relative source file path. Pair with the repo's commit SHA to deep-link " +
				"to the exact line in the IaC file that caused the finding.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "List failed controls in a repo scan report",
				Body:  "```\narmoctl repo-posture failed-controls --report-guid <guid>\n```",
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
