package integrations

import (
	"fmt"

	"github.com/spf13/cobra"
)

func FieldsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fields",
		Short: "Print the integrations resource cheatsheet",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			cheat := Cheatsheet()
			for scope, fields := range cheat {
				fmt.Fprintf(out, "### %s\n", scope)
				for _, f := range fields {
					fmt.Fprintf(out, "  %-22s %s\n", f.Name, f.Doc)
				}
			}
			return nil
		},
	}
}
