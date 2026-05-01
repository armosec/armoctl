package risks

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func SeveritiesCmd(clientFor cliclient.ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "severities",
		Short: "Aggregate risks by severity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			var obj map[string]any
			if err := cli.PostJSON(cmd.Context(), "/securityrisks/severities", nil, map[string]any{}, &obj); err != nil {
				return err
			}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: obj}, cliflags.OutputOptions(cmd, nil))
		},
	}
}
