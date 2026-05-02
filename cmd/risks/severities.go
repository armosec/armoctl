package risks

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func SeveritiesCmd(clientFor cliclient.ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "severities",
		Short: "Aggregate risks by severity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			res, err := cli.ListPaged(cmd.Context(), "/securityrisks/severities", nil, apiclient.ListOpts{
				Method:   "POST",
				Body:     map[string]any{},
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
}
