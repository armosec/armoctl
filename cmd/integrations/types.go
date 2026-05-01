package integrations

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
