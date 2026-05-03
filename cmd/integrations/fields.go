package integrations

import (
	"fmt"

	"github.com/spf13/cobra"
)

func FieldsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fields",
		Short: "Print the integrations resource cheatsheet",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			cheat := Cheatsheet()
			for scope, fields := range cheat {
				_, _ = fmt.Fprintf(out, "### %s\n", scope)
				for _, f := range fields {
					_, _ = fmt.Fprintf(out, "  %-22s %s\n", f.Name, f.Doc)
				}
			}
			return nil
		},
	}
}
