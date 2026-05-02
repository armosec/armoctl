package cloudaccounts

import (
	"fmt"

	"github.com/spf13/cobra"
)

func FieldsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fields",
		Short: "Print the ECS cluster resource cheatsheet",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "### ecs\n")
			for _, f := range Cheatsheet() {
				_, _ = fmt.Fprintf(out, "  %-22s %s\n", f.Name, f.Doc)
			}
			return nil
		},
	}
}
