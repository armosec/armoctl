package incidents

import (
	"net/url"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func AlertsCmd(clientFor ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "alerts [guid]",
		Short: "List alerts grouped under one incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "alerts requires an incident GUID"}
			}
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			path := "/runtime/incidents/" + args[0] + "/alerts/list"
			res, err := cli.ListPaged(cmd.Context(), path, url.Values{}, apiclient.ListOpts{
				Limit: pg.Limit, Page: pg.Page, PageSize: pg.PageSize,
			})
			if err != nil {
				return err
			}
			list := output.List{Items: res.Items, Total: res.Total, Page: res.Page, PageSize: res.PageSize, NextCursor: res.NextCursor}
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, nil))
		},
	}
}
