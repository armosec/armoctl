package repoposture

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func FilesCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "files",
		Short: "List files",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			body := map[string]any{}
			if repo, _ := cmd.Flags().GetString("repository"); repo != "" {
				body["innerFilters"] = []map[string]string{{"repoName": repo}}
			}
			res, err := cli.ListPaged(cmd.Context(), "/repositoryPosture/files", nil, apiclient.ListOpts{
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
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, FileSummary))
		},
	}
	c.Flags().String("repository", "", "Filter by repository name")
	return c
}
