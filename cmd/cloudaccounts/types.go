package cloudaccounts

import "github.com/armosec/armoctl/internal/clierr"

// codeForStatus returns the appropriate clierr.Code for an HTTP status code.
func codeForStatus(s int) clierr.Code {
	switch {
	case s == 401, s == 403:
		return clierr.CodeAuth
	case s == 404:
		return clierr.CodeNotFound
	case s == 409:
		return clierr.CodeConflict
	case s >= 400 && s < 500:
		return clierr.CodeBadInput
	default:
		return clierr.CodeServer
	}
}

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
