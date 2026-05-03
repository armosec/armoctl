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

	exc := &cobra.Command{Use: "exceptions", Short: "Manage security-risk exception policies (risk acceptance)"}
	exc.AddCommand(ExceptionsListCmd(clientFor))
	exc.AddCommand(ExceptionsGetCmd(clientFor))
	exc.AddCommand(ExceptionsCreateCmd(clientFor))
	exc.AddCommand(ExceptionsUpdateCmd(clientFor))
	exc.AddCommand(ExceptionsDeleteCmd(clientFor))
	c.AddCommand(exc)

	return c
}
