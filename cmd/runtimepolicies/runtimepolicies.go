// Package runtimepolicies implements the `armoctl runtime-policies` cluster.
package runtimepolicies

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "runtime-policies", Short: "Manage runtime policies"}
	c.AddCommand(FieldsCmd())
	c.AddCommand(ListCmd(clientFor))
	c.AddCommand(CreateCmd(clientFor))
	c.AddCommand(UpdateCmd(clientFor))
	return c
}
