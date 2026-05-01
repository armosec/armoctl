package vulns

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/spf13/cobra"
)

// Cmd builds the `armoctl vulns` cluster root.
// Subsequent tasks add list/aggregate/exception subcommands here.
func Cmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "vulns", Short: "Inspect and manage vulnerabilities"}
	c.AddCommand(FieldsCmd())
	// list/aggregate/exceptions/scan subcommands added in subsequent tasks.
	return c
}
