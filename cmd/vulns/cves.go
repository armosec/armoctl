package vulns

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func CVEsCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "cves",
		Short: "List vulnerabilities (CVEs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			body := map[string]any{}
			if sev, _ := cmd.Flags().GetString("severity"); sev != "" {
				body["innerFilters"] = []map[string]string{{"severity": sev}}
			}
			res, err := cli.ListPaged(cmd.Context(), "/vulnerability_v2/vulnerability/list", nil, apiclient.ListOpts{
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
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, CVESummary))
		},
	}
	c.Flags().String("severity", "", "Filter by severity (Critical|High|Medium|Low)")
	return c
}
