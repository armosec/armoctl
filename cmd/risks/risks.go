package risks

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "risks", Short: "Inspect prioritized security risks"}
	c.AddCommand(FieldsCmd())
	c.AddCommand(ListCmd(clientFor))
	c.AddCommand(ResourcesCmd(clientFor))
	c.AddCommand(SeveritiesCmd(clientFor))
	return c
}
