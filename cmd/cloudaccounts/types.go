package cloudaccounts

var ECSSummary = []string{"clusterARN", "name", "region", "accountID", "status", "lastSeen"}

type Field struct {
	Name string
	Doc  string
}

func Cheatsheet() []Field {
	return []Field{
		{"clusterARN", "AWS ECS cluster ARN."},
		{"name", "Cluster name."},
		{"region", "AWS region."},
		{"accountID", "AWS account ID."},
		{"status", "Connection status."},
		{"lastSeen", "RFC3339 last-seen time."},
	}
}
