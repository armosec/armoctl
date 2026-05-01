package attackchains

import (
	"fmt"

	"github.com/spf13/cobra"
)

// FieldsCmd is `armoctl attack-chains fields`.
func FieldsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fields",
		Short: "Print the attack chains resource cheatsheet",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Default summary fields (use --full or --fields to override):")
			for _, f := range ChainSummary {
				fmt.Fprintf(out, "  %s\n", f)
			}
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Field cheatsheet:")
			for _, f := range Cheatsheet() {
				fmt.Fprintf(out, "  %-22s %s\n", f.Name, f.Doc)
			}
			return nil
		},
	}
}
