package cloudaccounts

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-cloud-accounts",
		Cluster: "cloud-accounts",
		Description: "Cloud account onboarding — list/connect/disconnect ECS accounts. Use to see which " +
			"AWS accounts ARMO is monitoring or to onboard a new one.",
		Summary: "Cloud accounts is the AWS-side onboarding surface. Today it covers ECS account " +
			"connection state; future cloud surfaces (EKS, GCP) will land here.",
		FieldNotes: map[string]string{
			"status": "Connection status: connected / pending / failed. 'pending' means CloudFormation " +
				"rollout is still in progress.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "List ECS accounts",
				Body:  "```\narmoctl cloud-accounts ecs list\n```",
			},
		},
	})
}

func convertCheatsheet(in []Field) map[string][]skillmeta.Field {
	fs := make([]skillmeta.Field, len(in))
	for i, f := range in {
		fs[i] = skillmeta.Field{Name: f.Name, Doc: f.Doc}
	}
	return map[string][]skillmeta.Field{"ecs": fs}
}
