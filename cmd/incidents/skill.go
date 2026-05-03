package incidents

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-incidents",
		Cluster: "incidents",
		Description: "ARMO runtime incidents — list active threats, fetch alerts on a single incident, " +
			"explain an incident's signal, resolve/silence incidents. Use when investigating live runtime " +
			"alerts or post-mortems.",
		Summary: "The incidents cluster is the live runtime-threat surface. An incident is the unit of " +
			"triage; it bundles many alerts produced from runtime detection rules. Severity is " +
			"ARMO-policy-adjusted, not raw alert severity. Use 'incidents alerts <guid>' to get the full " +
			"alert payload behind an incident before resolving it.",
		FieldNotes: map[string]string{
			"attributes.incidentStatus": "Live state machine: open → investigating → resolved. " +
				"Access with path syntax: .attributes.incidentStatus",
			"signature": "Unique fingerprint identifying the rule that fired. Incidents sharing a " +
				"signature are the same detection event pattern.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "List Critical open incidents",
				Body:  "```\narmoctl incidents list --severity Critical\n```",
			},
			{
				Title: "Get all alerts for an incident",
				Body:  "```\narmoctl incidents alerts <incident-guid>\n```",
			},
		},
	})
}

func convertCheatsheet(in []Field) map[string][]skillmeta.Field {
	fs := make([]skillmeta.Field, len(in))
	for i, f := range in {
		fs[i] = skillmeta.Field{Name: f.Name, Doc: f.Doc}
	}
	return map[string][]skillmeta.Field{"incidents": fs}
}
