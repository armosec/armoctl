package runtimerules

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

// GetCmd builds `armoctl runtime-rules get [ruleGUID]`.
func GetCmd(clientFor cliclient.ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "get [ruleGUID]",
		Short: "Get a single runtime rule by GUID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "get requires a rule GUID"}
			}
			cli := clientFor(cmd)
			path := "/runtime/rules/" + args[0]
			var obj map[string]any
			if err := cli.GetJSON(cmd.Context(), path, nil, &obj); err != nil {
				return err
			}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: obj}, cliflags.OutputOptions(cmd, nil))
		},
	}
}
