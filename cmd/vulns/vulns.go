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
	c.AddCommand(WorkloadsCmd(clientFor))
	c.AddCommand(ImagesCmd(clientFor))
	c.AddCommand(ComponentsCmd(clientFor))
	c.AddCommand(CVEsCmd(clientFor))
	c.AddCommand(HostsCmd(clientFor))
	c.AddCommand(TopCmd(clientFor))
	c.AddCommand(SeverityCmd(clientFor))
	c.AddCommand(HistoryCmd(clientFor))
	c.AddCommand(ScanCmd(clientFor))

	exc := &cobra.Command{Use: "exceptions", Short: "Vulnerability exception policies"}
	exc.AddCommand(ExceptionsListCmd(clientFor))
	exc.AddCommand(ExceptionsCreateCmd(clientFor))
	exc.AddCommand(ExceptionsUpdateCmd(clientFor))
	exc.AddCommand(ExceptionsDeleteCmd(clientFor))
	c.AddCommand(exc)

	return c
}
