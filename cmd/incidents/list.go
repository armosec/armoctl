package incidents

import (
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

// ClientFor returns the apiclient configured for the running command.
// Cluster commands take this as a function so tests can inject stubs.
type ClientFor func(cmd *cobra.Command) *apiclient.Client

// ListCmd builds `armoctl incidents list`.
func ListCmd(clientFor ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List runtime incidents",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			body := map[string]any{}
			if sev, _ := cmd.Flags().GetString("severity"); sev != "" {
				body["severity"] = sev
			}
			res, err := cli.ListPaged(cmd.Context(), "/runtime/incidents", nil, apiclient.ListOpts{
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
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, SummaryFields))
		},
	}
	c.Flags().String("severity", "", "Filter by severity (critical|high|medium|low)")
	return c
}
