package attackchains

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "attack-chains", Short: "Inspect attack chains"}
	c.AddCommand(FieldsCmd())
	c.AddCommand(ListCmd(clientFor))
	return c
}
