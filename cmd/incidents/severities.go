package incidents

import (
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func SeveritiesCmd(clientFor ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "severities",
		Short: "Get aggregate incident counts per severity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			var obj map[string]any
			if err := cli.GetJSON(cmd.Context(), "/runtime/incidentsPerSeverity", nil, &obj); err != nil {
				return err
			}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: obj}, cliflags.OutputOptions(cmd, nil))
		},
	}
}
