package integrations

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

var JiraProjectSummary = []string{"key", "name", "id", "projectTypeKey", "lead"}
var JiraIssueTypeSummary = []string{"id", "name", "description", "subtask"}

type Field struct {
	Name string
	Doc  string
}

func Cheatsheet() map[string][]Field {
	return map[string][]Field{
		"jira-projects": {
			{"key", "Project key (e.g. SEC)."},
			{"name", "Human-readable project name."},
			{"id", "Numeric Jira project ID."},
			{"projectTypeKey", "software | service_desk | business."},
			{"lead", "Project lead identifier."},
		},
		"jira-issue-types": {
			{"id", "Issue type ID."},
			{"name", "Issue type name (Bug, Task, Epic, ...)."},
			{"description", "Description."},
			{"subtask", "Whether this is a subtask type."},
		},
	}
}
