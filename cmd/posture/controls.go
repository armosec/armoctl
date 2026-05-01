package posture

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func ControlsCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "controls",
		Short: "List posture controls",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			body := map[string]any{}
			if fw, _ := cmd.Flags().GetString("framework"); fw != "" {
				body["innerFilters"] = []map[string]string{{"frameworkName": fw}}
			}
			res, err := cli.ListPaged(cmd.Context(), "/posture/controls", nil, apiclient.ListOpts{
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
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, ControlSummary))
		},
	}
	c.Flags().String("framework", "", "Filter by framework name")
	return c
}
