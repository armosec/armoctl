package cloudaccounts

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "cloud-accounts", Short: "Manage cloud accounts (ECS)"}
	c.AddCommand(FieldsCmd())
	ecs := &cobra.Command{Use: "ecs", Short: "ECS cluster connections"}
	ecs.AddCommand(ECSListCmd(clientFor))
	ecs.AddCommand(ECSConnectCmd(clientFor))
	ecs.AddCommand(ECSDisconnectCmd(clientFor))
	c.AddCommand(ecs)
	return c
}
