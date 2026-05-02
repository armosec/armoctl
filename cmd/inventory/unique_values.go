package inventory

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func UniqueValuesCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "unique-values [field]",
		Short: "Get unique values for an inventory field",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			fieldName := args[0]
			body := map[string]any{"fields": map[string]any{fieldName: ""}}
			var obj map[string]any
			if err := cli.PostJSON(cmd.Context(), "/uniqueValues/inventory", nil, body, &obj); err != nil {
				return err
			}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: obj}, cliflags.OutputOptions(cmd, nil))
		},
	}
	return c
}
