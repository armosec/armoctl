package runtimerules

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

// ListCmd builds `armoctl runtime-rules list`.
func ListCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List runtime rules",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			body := map[string]any{}
			if name, _ := cmd.Flags().GetString("name"); name != "" {
				body["innerFilters"] = []map[string]any{
					{"name": name},
				}
			}
			res, err := cli.ListPaged(cmd.Context(), "/runtime/rules/list", nil, apiclient.ListOpts{
				Limit: pg.Limit, Page: pg.Page, PageSize: pg.PageSize,
				Method: "POST", Body: body,
			})
			if err != nil {
				return err
			}
			list := output.List{
				Items: res.Items, Total: res.Total,
				Page: res.Page, PageSize: res.PageSize, NextCursor: res.NextCursor,
			}
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, RuleSummary))
		},
	}
	c.Flags().String("name", "", "Filter by rule name")
	return c
}
