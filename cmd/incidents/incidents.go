package incidents

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "incidents", Short: "Inspect and manage runtime incidents"}
	c.AddCommand(FieldsCmd())
	c.AddCommand(ListCmd(clientFor))
	c.AddCommand(AlertsCmd(clientFor))
	c.AddCommand(ExplainCmd(clientFor))
	c.AddCommand(ResolveCmd(clientFor))
	c.AddCommand(SeveritiesCmd(clientFor))
	return c
}
