package incidents

import (
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func ExplainCmd(clientFor ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "explain [guid]",
		Short: "Get the platform's explanation for an incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "explain requires a GUID"}
			}
			cli := clientFor(cmd)
			var obj map[string]any
			if err := cli.GetJSON(cmd.Context(), "/runtime/incidents/"+args[0]+"/explain", nil, &obj); err != nil {
				return err
			}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: obj}, cliflags.OutputOptions(cmd, nil))
		},
	}
}
