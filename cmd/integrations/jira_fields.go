package integrations

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func JiraFieldsCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "fields",
		Short: "List Jira fields available for an issue type",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectKey, _ := cmd.Flags().GetString("project")
			issueTypeID, _ := cmd.Flags().GetString("issue-type")
			if projectKey == "" || issueTypeID == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "fields requires --project and --issue-type"}
			}
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			body := map[string]any{
				"projectKey": projectKey,
				"issueTypeID": issueTypeID,
			}
			res, err := cli.ListPaged(cmd.Context(), "/integrations/jira/issueTypesV2/fields/search", nil, apiclient.ListOpts{
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
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, nil))
		},
	}
	c.Flags().String("project", "", "Project key (required)")
	c.Flags().String("issue-type", "", "Issue type ID (required)")
	return c
}
