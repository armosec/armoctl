package seccomp

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-seccomp",
		Cluster: "seccomp",
		Description: "Generated seccomp profiles — list and generate profiles per workload. Use to restrict " +
			"syscalls to those observed at runtime.",
		Summary: "Same model as network-policies but for seccomp: ARMO records the syscall set at runtime " +
			"and emits a tight allow-list profile.",
		FieldNotes: map[string]string{
			"containerName": "Container the profile applies to. Profiles are container-level, which is " +
				"more precise than pod-level but requires one profile per container.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "Generate a seccomp profile for a workload",
				Body:  "```\narmoctl seccomp generate --wlid <workload-id>\n```",
			},
		},
	})
}

func convertCheatsheet(in []Field) map[string][]skillmeta.Field {
	fs := make([]skillmeta.Field, len(in))
	for i, f := range in {
		fs[i] = skillmeta.Field{Name: f.Name, Doc: f.Doc}
	}
	return map[string][]skillmeta.Field{"profiles": fs}
}
