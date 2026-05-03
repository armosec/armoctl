package attackchains

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

// attackChainsResponse mirrors the live envelope:
// { "total": {...}, "response": { "attackChains": [...] }, "cursor": "" }
type attackChainsResponse struct {
	Response struct {
		AttackChains []any `json:"attackChains"`
	} `json:"response"`
	Total struct {
		Value int `json:"value"`
	} `json:"total"`
}

func ListCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List attack chains",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			body := map[string]any{
				"pageNum":  pg.Page,
				"pageSize": pg.PageSize,
			}
			var r attackChainsResponse
			if err := cli.PostJSON(cmd.Context(), "/attackchains", nil, body, &r); err != nil {
				return err
			}
			list := output.List{Items: r.Response.AttackChains, Total: r.Total.Value}
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, ChainSummary))
		},
	}
	return c
}
