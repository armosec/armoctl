package vulns

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func TopCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "top",
		Short: "Top vulnerabilities (used by the weekly report)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			body := map[string]any{}
			if sev, _ := cmd.Flags().GetString("severity"); sev != "" {
				body["innerFilters"] = []map[string]string{{"severity": sev}}
			}
			res, err := cli.ListPaged(cmd.Context(), "/vulnerability/topVulnerabilities", nil, apiclient.ListOpts{
				Method: "POST", Body: body,
				Limit: pg.Limit, Page: pg.Page, PageSize: pg.PageSize,
			})
			if err != nil {
				return err
			}
			list := output.List{Items: res.Items, Total: res.Total, Page: res.Page, PageSize: res.PageSize, NextCursor: res.NextCursor}
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, nil))
		},
	}
	c.Flags().String("severity", "", "Filter by severity")
	return c
}
