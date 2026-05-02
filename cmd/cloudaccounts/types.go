package cloudaccounts

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/armosec/armoctl/internal/clierr"
)

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

// extractAPIMessage mirrors apiclient.mapHTTPError's body extraction so cloud-account
// commands that bypass apiclient.decode still surface the same human-readable error
// text as the rest of the CLI.
func extractAPIMessage(body []byte, status int) string {
	var msg struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	_ = json.Unmarshal(body, &msg)
	if m := msg.Message; m != "" {
		return m
	}
	if m := msg.Error; m != "" {
		return m
	}
	if m := strings.TrimSpace(string(body)); m != "" {
		return m
	}
	return http.StatusText(status)
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
