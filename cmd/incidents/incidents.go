package incidents

import "github.com/spf13/cobra"

// Cmd builds the `armoctl incidents` cluster root.
func Cmd() *cobra.Command {
	c := &cobra.Command{Use: "incidents", Short: "Inspect and manage runtime incidents"}
	c.AddCommand(FieldsCmd())
	// list/get/alerts/explain/resolve/unresolve/severities are added in subsequent tasks.
	return c
}
