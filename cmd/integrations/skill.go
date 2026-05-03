package integrations

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-integrations",
		Cluster: "integrations",
		Description: "Outbound integrations — alert channels (Slack/email/webhook), SIEM forwarders, " +
			"Jira ticket creation. Use to wire ARMO into external workflows.",
		Summary: "Integrations is where ARMO emits, not consumes. Alert channels deliver events; SIEM " +
			"forwarders ship logs; Jira lets the agent open tickets directly.",
		FieldNotes: map[string]string{
			"projectTypeKey": "Jira project type: software | service_desk | business. Determines which " +
				"issue types and workflows are available in the project.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "Create a Jira ticket from an incident",
				Body: "```\narmoctl integrations jira create-ticket --project <key> " +
					"--issue-type Bug --summary 'Incident <guid>'\n```",
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
