package repoposture

import (
	"fmt"
	"sort"

	"github.com/armosec/armoctl/internal/clierr"
	"github.com/spf13/cobra"
)

// FieldsCmd is `armoctl repo-posture fields [scope]`.
//
// With no argument, prints all scopes' cheatsheets.
// With a scope, prints just that scope.
func FieldsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fields [scope]",
		Short: "Print the repo-posture resource cheatsheet (optionally filtered by scope)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			cs := Cheatsheet()
			scopes := make([]string, 0, len(cs))
			for s := range cs {
				scopes = append(scopes, s)
			}
			sort.Strings(scopes)

			if len(args) == 1 {
				want := args[0]
				fields, ok := cs[want]
				if !ok {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: fmt.Sprintf("unknown scope %q (have %v)", want, scopes)}
				}
				printScope(out, want, fields)
				return nil
			}
			for _, s := range scopes {
				printScope(out, s, cs[s])
				_, _ = fmt.Fprintln(out)
			}
			return nil
		},
	}
}

func printScope(out interface{ Write(p []byte) (int, error) }, scope string, fields []Field) {
	_, _ = fmt.Fprintf(out, "### %s\n", scope)
	for _, f := range fields {
		_, _ = fmt.Fprintf(out, "  %-22s %s\n", f.Name, f.Doc)
	}
}
