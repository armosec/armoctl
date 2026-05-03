package vulns

import (
	"fmt"
	"sort"

	"github.com/armosec/armoctl/internal/clierr"
	"github.com/spf13/cobra"
)

// FieldsCmd is `armoctl vulns fields [scope]`.
//
// With no argument, prints all scopes' cheatsheets.
// With a scope, prints just that scope.
func FieldsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fields [scope]",
		Short: "Print the vulns resource cheatsheet (optionally filtered by scope)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: fmt.Sprintf("fields takes at most one scope (got %d)", len(args))}
			}
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
