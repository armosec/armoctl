package integrations

import (
	"context"
	"os"
	"strings"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func JiraCreateTicketCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "create-ticket",
		Short: "Create a Jira issue (ticket)",
		RunE: func(cmd *cobra.Command, args []string) error {
			project, _ := cmd.Flags().GetString("project")
			issueType, _ := cmd.Flags().GetString("issue-type")
			summary, _ := cmd.Flags().GetString("summary")
			description, _ := cmd.Flags().GetString("description")
			extraFields, _ := cmd.Flags().GetStringSlice("field")

			if project == "" || issueType == "" || summary == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create-ticket requires --project, --issue-type, and --summary"}
			}

			fields := map[string]any{
				"summary": summary,
			}
			if description != "" {
				fields["description"] = description
			}
			for _, kv := range extraFields {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) == 2 {
					fields[parts[0]] = parts[1]
				}
			}

			body := map[string]any{
				"projectKey":   project,
				"issueTypeName": issueType,
				"fields":       fields,
			}

			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "integrations.jira.create-ticket",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": "/integrations/jira/issueV2", "body": body},
				ArgsLog: "project=" + project + " issueType=" + issueType + " summary=" + summary,
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					var result any
					if err := cli.PostJSON(ctx, "/integrations/jira/issueV2", nil, body, &result); err != nil {
						return nil, safety.ExecMeta{}, err
					}
					return result, safety.ExecMeta{URL: "POST /integrations/jira/issueV2", Status: 200}, nil
				},
			})
		},
	}
	c.Flags().String("project", "", "Project key (required)")
	c.Flags().String("issue-type", "", "Issue type name (required)")
	c.Flags().String("summary", "", "Issue summary (required)")
	c.Flags().String("description", "", "Issue description (optional)")
	c.Flags().StringSlice("field", []string{}, "Extra fields as key=value (repeatable, optional)")
	return c
}
