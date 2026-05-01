package posture

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func ExceptionsListCmd(clientFor cliclient.ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List posture exception policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			var raw []any
			if err := cli.GetJSON(cmd.Context(), "/postureExceptionPolicy", nil, &raw); err != nil {
				return err
			}
			list := output.List{Items: raw, Total: len(raw)}
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, ExceptionSummary))
		},
	}
}
