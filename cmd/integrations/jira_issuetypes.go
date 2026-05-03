package integrations

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func JiraIssueTypesCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "issue-types",
		Short: "List Jira issue types",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			projectKey, _ := cmd.Flags().GetString("project")
			body := map[string]any{}
			if projectKey != "" {
				body["projectKey"] = projectKey
			}
			res, err := cli.ListPaged(cmd.Context(), "/integrations/jira/issueTypesV2/search", nil, apiclient.ListOpts{
				Method:   "POST",
				Body:     body,
				Limit:    pg.Limit,
				Page:     pg.Page,
				PageSize: pg.PageSize,
			})
			if err != nil {
				return err
			}
			list := output.List{Items: res.Items, Total: res.Total, Page: res.Page, PageSize: res.PageSize, NextCursor: res.NextCursor}
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, JiraIssueTypeSummary))
		},
	}
	c.Flags().String("project", "", "Filter by project key")
	return c
}
