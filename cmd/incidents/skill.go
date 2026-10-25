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
			"attributes.incidentStatus": "Live state machine. Canonical values: " +
				"Open | Investigating | Dismissed | Resolved. " +
				"Access with path syntax: .attributes.incidentStatus",
			"signature": "Unique fingerprint identifying the rule that fired. Incidents sharing a " +
				"signature are the same detection event pattern.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "List Critical incidents (all statuses)",
				Body:  "```\narmoctl incidents list --severity Critical\n```",
			},
			{
				Title: "Get all alerts for an incident",
				Body:  "```\narmoctl incidents alerts <incident-guid>\n```",
			},
			{
				Title: "Dismiss incidents matching a filter",
				Body: "Preview, show the user, and wait for an explicit yes (see Mutations):\n\n" +
					"```\narmoctl incidents set-status --status Dismissed --filter severity=Low --dry-run\n```\n\n" +
					"After the user confirms, run the same command with `--yes` in place of `--dry-run`.",
			},
			{
				Title: "Bulk dismiss a list of incident GUIDs from stdin",
				Body: "Save the GUIDs to a file first, so the dry-run and the confirmed run act on the same list:\n\n" +
					"```\narmoctl incidents list --severity Low | jq -r '.items[].guid' > guids.txt\n" +
					"armoctl incidents set-status --status Dismissed --stdin --dry-run < guids.txt\n```\n\n" +
					"Show the user the dry-run and wait for an explicit yes. Then:\n\n" +
					"```\narmoctl incidents set-status --status Dismissed --stdin --yes < guids.txt\n```",
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
