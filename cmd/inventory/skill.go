package inventory

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-inventory",
		Cluster: "inventory",
		Description: "Cluster inventory — list workloads, get unique values for a field. Use to enumerate " +
			"or pivot on resources before applying another command.",
		Summary: "Inventory is the index of everything ARMO has seen. Use 'inventory list' to enumerate " +
			"workloads/resources and 'inventory unique-values <field>' to discover the legal values for a " +
			"given field (clusters, namespaces, kinds, etc.).",
		FieldNotes: map[string]string{
			"kind": "K8s kind — Deployment, StatefulSet, DaemonSet, Job, CronJob, Pod. Use " +
				"'inventory unique-values kind' to confirm the spelling expected by other commands.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "List unique namespaces in a cluster",
				Body:  "```\narmoctl inventory unique-values namespace\n```",
			},
		},
	})
}

func convertCheatsheet(in []Field) map[string][]skillmeta.Field {
	fs := make([]skillmeta.Field, len(in))
	for i, f := range in {
		fs[i] = skillmeta.Field{Name: f.Name, Doc: f.Doc}
	}
	return map[string][]skillmeta.Field{"inventory": fs}
}
