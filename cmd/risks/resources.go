package risks

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func ResourcesCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "resources",
		Short: "List resources affected by a security risk",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			riskID, _ := cmd.Flags().GetString("risk-id")
			body := map[string]any{
				"innerFilters": []map[string]string{{"securityRiskID": riskID}},
			}
			res, err := cli.ListPaged(cmd.Context(), "/securityrisks/resources", nil, apiclient.ListOpts{
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
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, ResourceSummary))
		},
	}
	c.Flags().String("risk-id", "", "Security risk ID to filter resources by (required)")
	_ = c.MarkFlagRequired("risk-id")
	return c
}
