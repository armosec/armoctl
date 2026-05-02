package runtimerules

import (
	"fmt"

	"github.com/spf13/cobra"
)

// FieldsCmd is `armoctl runtime-rules fields`.
func FieldsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fields",
		Short: "Print the runtime-rules resource cheatsheet",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintln(out, "Default summary fields (use --full or --fields to override):")
			for _, f := range RuleSummary {
				_, _ = fmt.Fprintf(out, "  %s\n", f)
			}
			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprintln(out, "Field cheatsheet:")
			for _, f := range Cheatsheet() {
				_, _ = fmt.Fprintf(out, "  %-22s %s\n", f.Name, f.Doc)
			}
			return nil
		},
	}
}
