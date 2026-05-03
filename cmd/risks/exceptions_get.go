package risks

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func ExceptionsGetCmd(clientFor cliclient.ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "get <guid>",
		Short: "Get a security-risk exception policy by GUID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "get requires the exception GUID"}
			}
			cli := clientFor(cmd)
			var raw map[string]any
			if err := cli.GetJSON(cmd.Context(), "/securityrisks/exceptions/"+args[0], nil, &raw); err != nil {
				return err
			}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: raw}, cliflags.OutputOptions(cmd, ExceptionSummary))
		},
	}
}
